/*
 * File: service.go
 * Project: api
 * File Created: Tuesday, 13th July 2021 2:05:16 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package api

import (
	"context"
	"net/http"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sagemakerruntime"
	"github.com/labstack/echo/v4"

	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
	config "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/config"
	realtime "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/sage/realtime"
)

var (
	ErrModelDeploymentNotFound = echo.NewHTTPError(http.StatusNotFound, "model deployment not in service")
	ErrModelNotTrained         = echo.NewHTTPError(http.StatusConflict, "model has not been succefully trained")
	ErrNoContent               = echo.NewHTTPError(http.StatusBadRequest, "no content to process")
)

// Service represents user application interface
type Service interface {
	RealtimeInference(echo.Context, realtimeInferenceReq) ([]realtime.DetectionReturn, error)
}

// Initialize initializes User application service with defaults
func Initialize(db *db.DB, platform *platform.Platform, cfg *config.Configuration) (*Model, error) {
	awsConfig, err := awsconfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	return &Model{
		db:                     db,
		platform:               platform,
		cfg:                    cfg,
		sagemakerRuntimeClient: sagemakerruntime.NewFromConfig(awsConfig),
	}, nil
}

// Model represents user application service
type Model struct {
	db                     *db.DB
	cfg                    *config.Configuration
	platform               *platform.Platform
	sagemakerRuntimeClient *sagemakerruntime.Client
}
