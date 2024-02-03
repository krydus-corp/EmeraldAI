/*
 * File: service.go
 * Project: model
 * File Created: Saturday, 24th July 2021 3:29:23 pm
 * Author: Anonymous (anonymous@gmail.com)
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package model

import (
	"net/http"

	"github.com/labstack/echo/v4"

	blob "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/blob"
	awssqs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/aws/sqs"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
	config "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/config"
)

var (
	ErrModelInvalidState           = echo.NewHTTPError(http.StatusConflict, "model in invalid state; can only be trained if not in 'Trained' or 'Training' state and if the associated dataset is not locked")
	ErrModelDeploymentInvalidState = echo.NewHTTPError(http.StatusConflict, "model deployment in invalid state; can only be deployed if not in 'In Service' or 'Creating' and only deleting when 'In Service'")
	ErrModelDeploymentNotFound     = echo.NewHTTPError(http.StatusNotFound, "model deployment not in service")
	ErrModelNotTrained             = echo.NewHTTPError(http.StatusConflict, "model has not been successfully trained")
	ErrInvalidAnnotationCount      = echo.NewHTTPError(http.StatusBadRequest, "annotations are less than required minimum of 10")
	ErrBatchBusy                   = echo.NewHTTPError(http.StatusConflict, "batch job already initialized or running")
	ErrMinimumClasses              = echo.NewHTTPError(http.StatusBadRequest, "classification project must contain at least 2 classes, each with at least 10 annotations")
)

// Initialize initializes Model application service with defaults
func Initialize(db *db.DB, platform *platform.Platform, blob *blob.Blob, cfg config.Application) (*Model, error) {

	trainPublisher, err := awssqs.NewPublisher(&awssqs.Config{}, cfg.TrainJobQueueName)
	if err != nil {
		return nil, err
	}
	endpointPublisher, err := awssqs.NewPublisher(&awssqs.Config{}, cfg.EndpointJobQueueName)
	if err != nil {
		return nil, err
	}
	batchPublisher, err := awssqs.NewPublisher(&awssqs.Config{}, cfg.BatchJobQueueName)
	if err != nil {
		return nil, err
	}
	garbagePublisher, err := awssqs.NewPublisher(&awssqs.Config{}, cfg.GarbageJobQueueName)
	if err != nil {
		return nil, err
	}

	return &Model{
		db:                db,
		platform:          platform,
		blob:              blob,
		trainPublisher:    trainPublisher,
		endpointPublisher: endpointPublisher,
		batchPublisher:    batchPublisher,
		garbagePublisher:  garbagePublisher}, nil
}

// Service represents Model application interface
type Service interface {
	Create(echo.Context, string, string, string, models.Preprocessors, models.Augmentations) (models.Model, error)
	View(echo.Context, string, string) (models.Model, error)
	List(echo.Context, string, string, models.Pagination) ([]models.Model, int64, error)
	Query(echo.Context, string, models.Query) ([]models.Model, int64, error)
	Update(echo.Context, Update) (models.Model, error)
	Delete(echo.Context, string, string) error

	Train(echo.Context, string, string) (models.Model, error)
	Deploy(echo.Context, string, string) error
	DeleteDeployment(echo.Context, string, string) error
	CreateBatch(echo.Context, string, string, int) error
}

// Model represents model application service
type Model struct {
	db       *db.DB
	platform *platform.Platform
	blob     *blob.Blob

	trainPublisher    awssqs.Publisher
	endpointPublisher awssqs.Publisher
	batchPublisher    awssqs.Publisher
	garbagePublisher  awssqs.Publisher
}
