/*
 * File: model.go
 * Project: model
 * File Created: Saturday, 24th July 2021 3:42:59 pm
 * Author: Anonymous (anonymous@gmail.com)
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package model

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"

	batchBL "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/worker/batch"
	endpointBL "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/worker/endpoint"
	garbageBL "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/worker/garbage"
	trainBL "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/worker/train"

	image "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/image"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	worker "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/worker"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

func (m Model) Create(c echo.Context, userid, projectid, name string, preprocessing models.Preprocessors, augmentation models.Augmentations) (models.Model, error) {

	// Get the original dataset associated with the project
	cursor, err := m.platform.DatasetDB.FindVersion(m.db, userid, projectid, 0)
	if err != nil {
		return models.Model{}, err
	}
	defer cursor.Close(context.TODO())

	datasets := []models.Dataset{}
	if err := cursor.All(context.TODO(), &datasets); err != nil {
		log.Errorf("Error retrieving project's dataset; project=%s err=%s", projectid, len(datasets), err.Error())
		return models.Model{}, err
	}

	if len(datasets) == 0 {
		return models.Model{}, platform.ErrDatasetDoesNotExist
	}

	if len(datasets) > 1 {
		log.Errorf("Unexpected number of datasets associated with project=%s; expecting 1 got %d", projectid, len(datasets))
		return models.Model{}, platform.ErrDatasetCorruptState
	}

	// Get project associated with model
	project, err := m.platform.ProjectDB.View(m.db, userid, projectid)
	if err != nil {
		return models.Model{}, err
	}

	// For classification projects, Check that there are at least 2 classes
	if project.AnnotationType == models.ProjectAnnotationTypeClassification.String() {
		if err := m.minimumClassCount(userid, projectid, datasets[0].ID.Hex()); err != nil {
			return models.Model{}, err
		}
	}

	// Create model
	model := models.NewModel(name, userid, projectid, datasets[0].ID.Hex(), m.blob.Bucket, preprocessing, augmentation, map[string]interface{}{"type": project.AnnotationType})
	if _, err = m.platform.ModelDB.Create(m.db, model); err != nil {
		return models.Model{}, err
	}

	return model, nil
}

func (m Model) View(c echo.Context, userid, modelid string) (models.Model, error) {
	model, err := m.platform.ModelDB.View(m.db, userid, modelid)
	if err != nil {
		return models.Model{}, err
	}

	return model, nil
}

func (m Model) List(c echo.Context, userid, projectid string, p models.Pagination) ([]models.Model, int64, error) {
	return m.platform.ModelDB.List(m.db, userid, projectid, p)
}

func (m Model) Query(ctx echo.Context, userid string, q models.Query) ([]models.Model, int64, error) {
	return m.platform.ModelDB.Query(m.db, userid, q)
}

// Update is a struct which contains model information used for updating.
type Update struct {
	ModelID       string
	UserID        string
	Name          string
	Parameters    models.TrainParameters
	Preprocessing models.Preprocessors
	Augmentation  models.Augmentations
}

func (m Model) Update(c echo.Context, r Update) (models.Model, error) {
	id, err := primitive.ObjectIDFromHex(r.ModelID)
	if err != nil {
		return models.Model{}, platform.ErrModelDoesNotExist
	}

	if err := m.platform.ModelDB.Update(m.db, models.Model{
		ID:            id,
		UserID:        r.UserID,
		Name:          r.Name,
		Parameters:    r.Parameters,
		Preprocessing: r.Preprocessing,
		Augmentation:  r.Augmentation,
	}); err != nil {
		return models.Model{}, err
	}

	return m.platform.ModelDB.View(m.db, r.UserID, r.ModelID)
}

func (m Model) Delete(c echo.Context, userid, modelid string) error {
	// Retrieve model for deployment info.
	model, err := m.platform.ModelDB.View(m.db, userid, modelid)
	if err != nil {
		return err
	}

	// If no deployment info then there should not be resources to remove
	// or garbage collector will clean up.
	if model.Deployment.ModelName != "" && model.Deployment.EndpointName != "" {
		// Queue up model delete job
		if _, err := worker.Retry(3, true, true, 2, func() (struct{}, error) {
			return struct{}{}, m.garbagePublisher.Publish(context.TODO(), garbageBL.NewEvent(modelid, model.Deployment.ModelName, model.Deployment.EndpointName, userid, garbageBL.ActionDelete))
		}); err != nil {
			log.Errorf("error queuing up model delete job; model=%s error=%s", modelid, err.Error())
			return err
		}
	}

	// Delete model from DB
	if err := m.platform.ModelDB.Delete(m.db, userid, modelid); err != nil {
		return err
	}

	// Delete ModelID from Exports associated with the model
	if err := m.platform.ExportDB.RemoveModelFromExports(m.db, userid, modelid); err != nil {
		return err
	}

	return nil
}

func (m Model) Train(ctx echo.Context, userid, modelid string) (models.Model, error) {
	var (
		model = models.Model{}
		err   error
	)

	// Retrieve model to train
	model, err = m.platform.ModelDB.View(m.db, userid, modelid)
	if err != nil {
		return model, err
	}

	// Retrieve associated dataset
	dataset, err := m.platform.DatasetDB.View(m.db, userid, model.DatasetID)
	if err != nil {
		return model, err
	}

	count, err := m.platform.AnnotationDB.CountAnnotations(m.db, userid, model.DatasetID)
	if err != nil {
		return model, errors.Wrapf(err, "error locating dataset=%s annotation count", model.DatasetID)
	}

	if *count < 10 {
		return model, ErrInvalidAnnotationCount
	}

	// Model cannot be in 'Trained' or 'Training' states and must not be locked
	if model.State == models.ModelStateTraining.String() || model.State == models.ModelStateTrained.String() || dataset.Locked {
		return model, ErrModelInvalidState
	}

	if err := m.platform.ModelDB.Update(m.db, models.Model{
		ID:             model.ID,
		UserID:         model.UserID,
		State:          models.ModelStateInitialized.String(), // reset state
		LastError:      aws.String(""),                        // reset error
		TrainStartedAt: time.Now(),
	}); err != nil {
		log.Errorf("error updating model=%s; error=%s", model.ID.Hex(), err.Error())
		return model, fmt.Errorf("error queuing up training job; unable to move model to 'ERR' state")
	}

	// Queue up training job
	if _, err := worker.Retry(3, true, true, 2, func() (struct{}, error) {
		return struct{}{}, m.trainPublisher.Publish(context.TODO(), trainBL.NewEvent(modelid, userid))
	}); err != nil {
		return model, err
	}

	return m.platform.ModelDB.View(m.db, userid, modelid)
}

func (m Model) Deploy(ctx echo.Context, userid, modelid string) error {

	// Retrieve model to train
	model, err := m.platform.ModelDB.View(m.db, userid, modelid)
	if err != nil {
		return err
	}

	// Model must have completed training and recieved a training job name
	if model.TrainingJobName == "" || model.State != models.ModelStateTrained.String() {
		return ErrModelNotTrained
	}

	// Model deployment can only be in UNKNOWN, ERR, or DELETED state to deploy
	status := models.DeploymentStatusFromString(model.Deployment.Status)
	if status > models.DeploymentStatusUnknown && status < models.DeploymentStatusDeleted {

		return ErrModelDeploymentInvalidState
	}

	if err := m.platform.ModelDB.Update(m.db, models.Model{
		ID:     model.ID,
		UserID: model.UserID,
		Deployment: models.Deployment{
			Status: models.DeploymentStatusInitialized.String(),
		},
	}); err != nil {
		log.Errorf("error updating model=%s; error=%s", modelid, err.Error())
		return err
	}

	// Queue up endpoint creation job
	if _, err := worker.Retry(3, true, true, 2, func() (struct{}, error) {
		return struct{}{}, m.endpointPublisher.Publish(context.TODO(), endpointBL.NewEvent(modelid, "", userid, endpointBL.ActionCreate))
	}); err != nil {
		log.Errorf("error queuing up endpoint creation job; model=%s error=%s", modelid, err.Error())
		return err
	}

	return nil
}

func (m Model) DeleteDeployment(ctx echo.Context, userid, modelid string) error {

	// Retrieve model to train
	model, err := m.platform.ModelDB.View(m.db, userid, modelid)
	if err != nil {
		return err
	}

	// Model deployment must exists
	if model.Deployment.EndpointName == "" {
		return ErrModelDeploymentNotFound
	}
	// Model deployment status must be in 'IN_SERVICE'state
	if model.Deployment.Status != models.DeploymentStatusInService.String() {
		return ErrModelDeploymentInvalidState
	}

	deployment := model.Deployment
	deployment.Status = models.DeploymentStatusDeleting.String()
	if err := m.platform.ModelDB.Update(m.db, models.Model{
		ID:         model.ID,
		UserID:     model.UserID,
		Deployment: deployment,
	}); err != nil {
		log.Errorf("error updating model=%s; error=%s", modelid, err.Error())
		return err
	}

	// Queue up endpoint delete job
	if _, err := worker.Retry(3, true, true, 2, func() (struct{}, error) {
		return struct{}{}, m.endpointPublisher.Publish(context.TODO(), endpointBL.NewEvent(modelid, model.Deployment.ModelName, userid, endpointBL.ActionDelete))
	}); err != nil {
		log.Errorf("error queuing up endpoint delete job; model=%s error=%s", modelid, err.Error())
		return err
	}

	return nil
}

func (m Model) CreateBatch(ctx echo.Context, userID, modelID string, thumbnailSize int) error {

	// Get model
	model, err := m.platform.ModelDB.View(m.db, userID, modelID)
	if err != nil {
		return err
	}

	// Model must be in 'Trained' state
	if model.State != models.ModelStateTrained.String() {
		return ErrModelNotTrained
	}
	// Model must have a deployment
	if model.Deployment.EndpointName == "" {
		return ErrModelDeploymentNotFound
	}
	// Check if batch job exist
	if model.Batch.Status == models.BatchStatusRunning.String() {
		return ErrBatchBusy
	}

	// Initialize batch
	if err := m.platform.ModelDB.Update(m.db, models.Model{
		ID:     model.ID,
		UserID: model.UserID,
		Batch:  models.NewBatch(),
	}); err != nil {
		log.Errorf("error updating model=%s; error=%s", modelID, err.Error())
		return err
	}

	if thumbnailSize == 0 {
		thumbnailSize = image.BoundingBoxDefaultThumbnailSize
	}

	// Queue up batch job
	if _, err := worker.Retry(3, true, true, 2, func() (struct{}, error) {
		return struct{}{}, m.batchPublisher.Publish(context.TODO(), batchBL.NewEvent(modelID, userID, thumbnailSize))
	}); err != nil {
		log.Errorf("error queuing up batch job; model=%s error=%s", modelID, err.Error())
		return err
	}

	return nil
}

func (m Model) minimumClassCount(userid, projectid, datasetid string) error {
	var results []struct {
		ID    string `bson:"_id"`
		Count int    `bson:"count"`
	}
	if err := m.platform.AnnotationDB.AnnotationsPerClass(m.db, userid, projectid, datasetid, &results); err != nil {
		log.Errorf("annotation per class err=%s", err.Error())
		return err
	}

	if len(results) < 2 {
		return ErrMinimumClasses
	}
	for _, class := range results {
		if class.Count < 10 {
			return ErrMinimumClasses
		}
	}
	return nil
}
