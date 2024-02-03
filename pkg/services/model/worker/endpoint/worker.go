/*
 * File: worker.go
 * Project: endpoint
 * File Created: Monday, 22nd August 2022 2:57:15 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package endpoint

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/pkg/errors"

	blob "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/blob"
	sqs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/aws/sqs"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
	modelConfig "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/config"
	endpoint "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/sage/endpoint"
)

type WorkerPool struct {
	Platform *platform.Platform
	DB       *db.DB
	Blob     *blob.Blob
	Config   modelConfig.Configuration

	sagemakerClient *sagemaker.Client
	consumer        sqs.Consumer
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
		WorkerPool:        cfg.ModelService.EndpointWorkerConfig.Concurrency,
		VisibilityTimeout: 30,
	}, queueName)
	if err != nil {
		return nil, err
	}

	awsConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	return &WorkerPool{
		Blob:     blob,
		DB:       db,
		Platform: platform,
		Config:   cfg,

		sagemakerClient: sagemaker.NewFromConfig(awsConfig),
		consumer:        consumer,
	}, nil
}

func (w *WorkerPool) Start() {
	// Start subscriber
	log.Infof("starting subscriber loop on queue=%s", w.Config.ModelService.EndpointJobQueueName)
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
	log.Debugf("Recieved message=%s", event.ToJSON())

	// Retrieve associated model
	model, err := w.Platform.ModelDB.View(w.DB, event.UserID, event.ModelID)
	if err != nil {
		log.Errorf("unable to view model asscoiated with train event; model=%s error=%s", event.ModelID, err.Error())
		return nil
	}

	endpoint, err := endpoint.New(&w.Config.ModelService.EndpointConfig, w.sagemakerClient, model.TrainingJobName)
	if err != nil {
		log.Errorf("unable to create model endpoint config; model=%s error=%s", event.ModelID, err.Error())
		return w.updateErrorState(err, &model)
	}

	// -------------- DELETE ENDPOINT EVENT -------------- //
	if event.Action == string(ActionDelete) {
		if model.Deployment.Status != models.DeploymentStatusDeleting.String() {
			// Should never reach this check - checked at the API layer
			log.Errorf("model deployment invalid state for DELETE action, must be in DELETING state; model=%s", event.ModelID)

			return nil
		}

		if err := endpoint.DeleteEndpoint(model.Deployment.EndpointName); err != nil { // error ignored intentionally - these will get garbage collected if failing to delete
			log.Errorf("error deleting model endpoint; model=%s error=%s", event.ModelID, err.Error())
			return nil
		}

		if err := w.Platform.ModelDB.Update(w.DB, models.Model{
			ID:     model.ID,
			UserID: model.UserID,
			Deployment: models.Deployment{
				Status:         models.DeploymentStatusDeleted.String(),
				ModelName:      event.ModelName,
				EndpointName:   "",
				EndpointCurl:   "",
				DeployedAt:     time.Time{},
				ExpireDuration: 0, // Never expire
				LastError:      aws.String(""),
			},
		}); err != nil {
			return w.updateErrorState(errors.Wrapf(err, "error updating model=%s after model deployment delete action", model.ID.Hex()), &model)
		}

		return nil
	}

	// -------------- CREATE EVENT -------------- //
	if model.Deployment.Status != models.DeploymentStatusInitialized.String() {
		// Should never reach this check - checked at the API layer
		log.Errorf("model deployment invalid state, must be in INITIALIZED, DELETED, or ERR state; model=%s", event.ModelID)

		return nil
	}

	if err := w.Platform.ModelDB.Update(w.DB, models.Model{
		ID:     model.ID,
		UserID: model.UserID,
		Deployment: models.Deployment{
			Status:         models.DeploymentStatusCreating.String(),
			ModelName:      "",
			EndpointName:   "",
			EndpointCurl:   "",
			DeployedAt:     time.Now(),
			ExpireDuration: 0, // Never expire
			LastError:      aws.String(""),
		},
	}); err != nil {
		return w.updateErrorState(errors.Wrapf(err, "error updating model=%s after model deployment success", model.ID.Hex()), &model)
	}

	if err := endpoint.CreateEndpoint(); err != nil {
		log.Errorf("unable to create model endpoint; model=%s error=%s", event.ModelID, err.Error())
		return w.updateErrorState(err, &model)
	}

	if err := endpoint.PollForStatus(); err != nil {
		return w.updateErrorState(err, &model)
	}

	// Update success status
	endpointName, _, modelName := endpoint.Describe()
	if err := w.Platform.ModelDB.Update(w.DB, models.Model{
		ID:     model.ID,
		UserID: model.UserID,
		Deployment: models.Deployment{
			Status:         models.DeploymentStatusInService.String(),
			ModelName:      modelName,
			EndpointName:   endpointName,
			EndpointCurl:   w.generateCurl(event.UserID, event.ModelID),
			DeployedAt:     time.Now(),
			ExpireDuration: 0, // Never expire
			LastError:      aws.String(""),
		},
	}); err != nil {
		return w.updateErrorState(errors.Wrapf(err, "error updating model=%s after model deployment success", model.ID.Hex()), &model)
	}

	return nil
}

// updateErrorState is a helper method for updating the deployment state after an error occurs
func (w *WorkerPool) updateErrorState(err error, model *models.Model) error {
	if err != nil {
		update := models.Model{
			ID:     model.ID,
			UserID: model.UserID,
			Deployment: models.Deployment{
				Status:         models.DeploymentStatusErr.String(),
				EndpointName:   "",
				EndpointCurl:   "",
				DeployedAt:     time.Time{},
				ExpireDuration: 0,
				LastError:      aws.String(err.Error()),
			},
		}
		if err := w.Platform.ModelDB.Update(w.DB, update); err != nil {
			log.Errorf("error updating model=%s after model deployment failure; err=%s", model.ID.Hex(), err.Error())
			return err
		}
	}
	return nil
}

func (w *WorkerPool) generateCurl(userid, modelid string) string {
	user, _ := w.Platform.UserDB.View(w.DB, userid)
	apiKey := user.APIKey
	return fmt.Sprintf("curl --location --request POST 'https://%s.emeraldai-dev.com/v1/models/%s/inference/realtime' --header 'userid: %s' --header 'apikey: %s' --header 'Content-Type: multipart/form-data' --form 'files=@/path/to/file'", os.Getenv("CLUSTER_ENV"), modelid, userid, apiKey)
}
