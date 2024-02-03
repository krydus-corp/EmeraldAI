/*
 * File: worker.go
 * Project: train
 * File Created: Tuesday, 16th August 2022 4:07:11 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package train

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"

	blob "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/blob"
	sqs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/aws/sqs"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	mail "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/mail"
	runtime "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/runtime"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
	modelConfig "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/config"
	sage "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/sage"
)

type WorkerPool struct {
	Platform *platform.Platform
	DB       *db.DB
	Blob     *blob.Blob
	Config   modelConfig.Configuration

	sagemakerClient *sagemaker.Client
	consumer        sqs.Consumer
	mail            *mail.SGModelMailService
}

func New(
	queueName string,
	blob *blob.Blob,
	db *db.DB,
	platform *platform.Platform,
	cfg modelConfig.Configuration,
) (*WorkerPool, error) {

	if queueName == "" {
		return nil, fmt.Errorf("queue name required")
	}

	consumer, err := sqs.NewConsumer(&sqs.Config{
		WorkerPool:        cfg.ModelService.TrainWorkerConfig.Concurrency,
		VisibilityTimeout: 30,
	}, queueName)
	if err != nil {
		return nil, err
	}

	awsConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	// Initialize SendGrid Mail
	sgAPIKey, err := runtime.GetEnv("SENDGRID_API_KEY")
	if err != nil {
		return nil, err
	}

	mail := mail.NewSGModelMailService(
		sgAPIKey,
		cfg.ModelService.TrainConfig.SendGrid.TrainingCompleteTemplateID,
		cfg.ModelService.TrainConfig.SendGrid.EmailSource,
		cfg.ModelService.TrainConfig.SendGrid.EmailSubject,
	)

	return &WorkerPool{
		Blob:     blob,
		DB:       db,
		Platform: platform,
		Config:   cfg,

		sagemakerClient: sagemaker.NewFromConfig(awsConfig),
		consumer:        consumer,
		mail:            mail,
	}, nil
}

func (w *WorkerPool) Start() {
	// Start subscriber
	log.Infof("starting subscriber loop on queue=%s", w.Config.ModelService.TrainJobQueueName)
	go w.consumer.Consume(w.callback)
}

// callback is a method for handling a train Event
// Any errors propagated up from this method will requeue the message for processed
func (w *WorkerPool) callback(msg *sqs.Message) error {
	// Decode the subscriber event
	var event Event
	if err := msg.Decode(&event); err != nil {
		log.Errorf("unable to decode train event; msg=%s error=%s", string(msg.Body()), err.Error())
		return nil
	}
	log.Debugf("Received message=%s", event.ToJSON())

	// Retrieve associated model
	model, err := w.Platform.ModelDB.View(w.DB, event.UserID, event.ModelID)
	if err != nil {
		log.Errorf("unable to view model associated with train event; model=%s error=%s", event.ModelID, err.Error())
		return nil
	}

	// Retrieve user
	user, err := w.Platform.UserDB.View(w.DB, model.UserID)
	if err != nil {
		return err
	}

	// Check duplicate sqs message
	if model.State != models.ModelStateInitialized.String() {
		return nil
	}

	// Move model to 'TRAINING' status
	if err := w.Platform.ModelDB.Update(w.DB, models.Model{
		ID:             model.ID,
		UserID:         model.UserID,
		State:          models.ModelStateTraining.String(),
		TrainStartedAt: time.Now(),
		LastError:      aws.String(""),
	}); err != nil {
		log.Errorf("error updating model=%s; error=%s", model.ID.Hex(), err.Error())
		return w.updateOnErrorState(err, &model, nil)
	}

	// Retrieve associated project
	project, err := w.Platform.ProjectDB.View(w.DB, event.UserID, model.ProjectID)
	if err != nil {
		log.Errorf("unable to view project associated with train event; project=%s error=%s", model.ProjectID, err.Error())
		return w.updateOnErrorState(err, &model, nil)
	}

	// Retrieve associated dataset
	datasetToTrain, err := w.Platform.DatasetDB.View(w.DB, event.UserID, model.DatasetID)
	if err != nil {
		log.Errorf("unable to view dataset associated with train event; project=%s error=%s", model.ProjectID, err.Error())
		return w.updateOnErrorState(err, &model, nil)
	}

	// Create a new version of the dataset and lock it
	versionedDataset, err := w.Platform.DatasetDB.Copy(w.DB, datasetToTrain, true, nil)
	if err != nil {
		log.Errorf("unable to version new dataset for training; model=%s dataset=%s error=%s", event.ModelID, datasetToTrain.ID.Hex(), err.Error())
		return w.updateOnErrorState(err, &model, nil)
	}

	// Prepare data, including generation of label integer map for this dataset
	labelIntegerMap, err := w.Platform.TagDB.TagIntegerMap(w.DB, versionedDataset.UserID, versionedDataset.ID.Hex())
	if err != nil {
		log.Errorf("unable to generate integer label map for dataset associated with train event; model=%s error=%s", event.ModelID, err.Error())
		return w.updateOnErrorState(err, &model, &versionedDataset.ID)
	}

	labelIntegerMapJson, _ := json.Marshal(labelIntegerMap) // intentionally ignored
	log.Debugf("Retrieved label integer map for dataset=%s; map=%s", versionedDataset.ID.Hex(), labelIntegerMapJson)

	log.Debugf("Starting preprocessing for dataset=%s", versionedDataset.ID.Hex())
	counts, err := w.preprocess(&model, versionedDataset, &project, labelIntegerMap)
	if err != nil {
		log.Errorf("error during data preprocessing for train event; model=%s error=%s", event.ModelID, err.Error())
		return w.updateOnErrorState(err, &model, &versionedDataset.ID)
	}
	log.Debugf("Preprocessing complete for dataset=%s; starting training on %d annotations", versionedDataset.ID.Hex(), counts.TrainCount)

	// Get max number of annotations in a single image for dataset to compute force padding label width
	var results []struct {
		MaxBoundingBoxes float64 `bson:"max_bounding_boxes"`
	}
	if project.AnnotationType == models.ProjectAnnotationTypeBoundingBox.String() {
		if err := w.Platform.AnnotationDB.MaxBoundingBoxesPerImage(w.DB, model.UserID, model.ProjectID, versionedDataset.ID.Hex(), &results); err != nil {
			log.Errorf("error computing force padding label width for train event; model=%s error=%s", event.ModelID, err.Error())
			return w.updateOnErrorState(err, &model, &versionedDataset.ID)
		}
		if len(results) != 1 {
			log.Errorf("unexpected result length when computing force padding label width for train event; model=%s error=%s", event.ModelID, err.Error())
			return w.updateOnErrorState(err, &model, &versionedDataset.ID)
		}
	}

	// Prepare config i.e. update dynamic settings
	cfg := w.Config.ModelService.TrainConfig
	cfg.OutputDataPath = model.Path
	cfg.NumClasses = int32(len(labelIntegerMap))
	cfg.NumTrainingSamples = int32(counts.TrainCount)
	cfg.NumValidationSamples = int32(counts.ValidationCount)
	// See https://docs.aws.amazon.com/sagemaker/latest/dg/object-detection-api-config.html for how ForcePaddingLabelWidth is computed
	if project.AnnotationType == models.ProjectAnnotationTypeBoundingBox.String() {
		cfg.ForcePaddingLabelWidth = int32(math.Max(350, float64(results[0].MaxBoundingBoxes)*5+2)) // 350 == default label_width
	}

	// Determine project type
	projectType, err := models.ProjectAnnotationTypeFromString(project.AnnotationType)
	if err != nil {
		log.Errorf("unable to determine train algorithm; project-type=%s error=%s", project.AnnotationType, err.Error())
		return w.updateOnErrorState(err, &model, &versionedDataset.ID)
	}

	// Train model
	trainingJobName, metrics, err := w.train(&cfg, projectType)
	if err != nil {
		log.Errorf("error during train; model=%s error=%s", model.ID.Hex(), err.Error())
		// Send training failed email notification
		if errSend := w.sendTrainingResultEmail(user.Username, user.Email, project.Name, model.Name, w.Config.ModelService.TrainConfig.SendGrid.TrainingFailedMessage); errSend != nil {
			log.Errorf("Unable to send training failed email to username=%s", user.Username)
		}
		return w.updateOnErrorState(err, &model, &versionedDataset.ID)
	}
	metrics["ObjectiveMetricName"] = sage.ObjectiveMetricName(projectType)

	// Dummy metrics
	if projectType == models.ProjectAnnotationTypeClassification {
		metrics["test:accuracy"] = "-"
		metrics["test:precision"] = "-"
		metrics["test:recall"] = "-"
	} else if projectType == models.ProjectAnnotationTypeBoundingBox {
		metrics["test:mAP"] = "-"
		metrics["test:precision"] = "-"
		metrics["test:recall"] = "-"
	}

	// Send training success email notification
	if err := w.sendTrainingResultEmail(user.Username, user.Email, project.Name, model.Name, w.Config.ModelService.TrainConfig.SendGrid.TrainingSuccessMessage); err != nil {
		log.Errorf("Unable to send training complete email to username=%s", user.Username)
	}

	// Update success status
	if err := w.Platform.ModelDB.Update(w.DB, models.Model{
		ID:              model.ID,
		UserID:          model.UserID,
		DatasetID:       versionedDataset.ID.Hex(), // update model with new versioned/locked dataset
		State:           models.ModelStateTrained.String(),
		TrainStartedAt:  time.Now(),
		LastError:       aws.String(""),
		Metrics:         metrics,
		IntegerMapping:  labelIntegerMap, // only update on success
		TrainingJobName: trainingJobName, // only update on success
	}); err != nil {
		return w.updateOnErrorState(errors.Wrapf(err, "error updating model=%s after train success", model.ID.Hex()), &model, &versionedDataset.ID)
	}

	// Update at user level
	var resourceMap map[string]interface{}
	resourceBytes, _ := json.Marshal(w.Config.ModelService.TrainConfig.Resource)
	json.Unmarshal(resourceBytes, &resourceMap)

	w.Platform.UserDB.AddUsage(w.DB, model.UserID, models.Usage{
		Time:          time.Now().UTC().Format(time.RFC3339),
		Type:          models.UsageTypeTrain,
		BillingMetric: models.BillingMetricSecond,
		BillableValue: float64(metrics["BillableTimeInSeconds"].(int32)),
		Metadata:      resourceMap,
	})

	return nil
}

// updateOnErrorState is a helper method for updating the model state after an error occurs
func (w *WorkerPool) updateOnErrorState(err error, model *models.Model, versionedDatasetId *primitive.ObjectID) error {

	// Update model lastError
	update := models.Model{
		ID:           model.ID,
		UserID:       model.UserID,
		State:        models.ModelStateErr.String(),
		TrainEndedAt: time.Now(),
		LastError:    aws.String(err.Error()),
	}
	if err := w.Platform.ModelDB.Update(w.DB, update); err != nil {
		log.Errorf("error updating model=%s after train failure; err=%s", model.ID.Hex(), err.Error())
		return err
	}

	if versionedDatasetId != nil {
		// Remove versioned dataset
		if err := w.Platform.DatasetDB.Delete(w.DB, *versionedDatasetId); err != nil {
			log.Errorf("error deleting versioned dataset after train failure; dataset=%s after train error; error=%s", versionedDatasetId.Hex(), err.Error())
			return err
		}
		// Delete versioned annotations
		if err := w.Platform.AnnotationDB.DeleteDatasetAnnotations(w.DB, model.UserID, model.ProjectID, versionedDatasetId.Hex()); err != nil {
			log.Errorf("error deleting versioned dataset annotations after train failure; dataset=%s after train error; error=%s", versionedDatasetId.Hex(), err.Error())
		}

		// Delete versioned tags.
		if err := w.Platform.TagDB.DeleteDatasetTags(w.DB, model.UserID, model.ProjectID, versionedDatasetId.Hex()); err != nil {
			log.Errorf("error deleting versioned dataset tags after train failure; dataset=%s after train error; error=%s", versionedDatasetId.Hex(), err.Error())
		}
	}

	return nil
}

func (w *WorkerPool) sendTrainingResultEmail(username, userEmail, projectName, modelName, status string) error {
	mailType := mail.TrainingConfirmation
	mailData := &mail.ModelMailData{
		Username: username,
		Status:   status,
		Project:  projectName,
		Model:    modelName,
	}
	mailReq := w.mail.NewMail(w.mail.TrainingCompleteEmail, []string{userEmail}, w.mail.TrainingCompleteSubject, mailType, mailData)
	err := w.mail.SendMail(mailReq)
	if err != nil {
		return err
	}
	return nil
}
