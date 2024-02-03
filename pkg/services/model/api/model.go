/*
 * File: model.go
 * Project: api
 * File Created: Tuesday, 16th August 2022 5:28:28 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package api

import (
	"path"
	"time"

	"github.com/labstack/echo/v4"

	stats "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/image"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	realtime "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/sage/realtime"
)

type realtimeInferenceReq struct {
	userID, modelID     string
	files               map[string][]byte
	octetStream         []byte
	confidenceThreshold float64
	heatmap             bool
}

func (m *Model) RealtimeInference(ctx echo.Context, req realtimeInferenceReq) ([]realtime.DetectionReturn, error) {

	// Get model
	model, err := m.platform.ModelDB.View(m.db, req.userID, req.modelID)
	if err != nil {
		return nil, err
	}

	// Model must be in 'Trained' state
	if model.State != models.ModelStateTrained.String() {
		return nil, ErrModelNotTrained
	}
	// Model must have a deployment
	if model.Deployment.EndpointName == "" {
		return nil, ErrModelDeploymentNotFound
	}

	results := []realtime.DetectionReturn{}
	totalInferenceTimeSeconds := float64(0)

	// Files provided
	if len(req.files) > 0 {
		for filename, filebytes := range req.files {
			imageStats, err := stats.GetStats(filebytes)
			if err != nil {
				return nil, err
			}

			start := time.Now()
			result, err := realtime.New(filebytes, model.Deployment.EndpointName, model.Metadata["type"].(string), m.sagemakerRuntimeClient)
			if err != nil {
				return nil, err
			}

			totalInferenceTimeSeconds += time.Since(start).Seconds()

			result.UpdateFilename(path.Base(filename))
			formattedResult := result.ToFormattedResult(model.IntegerMapping, req.confidenceThreshold, imageStats)
			results = append(results, formattedResult)
		}
	}

	// Octet stream
	if len(req.octetStream) > 0 {
		imageStats, err := stats.GetStats(req.octetStream)
		if err != nil {
			return nil, err
		}

		start := time.Now()
		result, err := realtime.New(req.octetStream, model.Deployment.EndpointName, model.Metadata["type"].(string), m.sagemakerRuntimeClient)
		if err != nil {
			return nil, err
		}

		totalInferenceTimeSeconds += time.Since(start).Seconds()

		formattedResult := result.ToFormattedResult(model.IntegerMapping, req.confidenceThreshold, imageStats)
		results = append(results, formattedResult)
	}

	// Confirm we have at least one of content IDs, files, or octet stream
	if len(results) == 0 {
		return nil, ErrNoContent
	}

	// Update usage - don't hold up the request for this update
	go func() {
		if err := m.updateUsage(req, len(results), totalInferenceTimeSeconds); err != nil {
			log.Errorf("unable to record endpoint usage for modelid=%s userid=%s; err=%s", req.modelID, req.userID, err.Error())
		}
	}()

	// TODO: Optionally, add heatmap

	return results, nil
}

func (m *Model) updateUsage(req realtimeInferenceReq, count int, inferenceTimeSeconds float64) error {
	return m.platform.UserDB.AddUsage(m.db, req.userID, models.Usage{
		Time:          time.Now().UTC().Format(time.RFC3339),
		Type:          models.UsageTypeEndpoint,
		BillingMetric: models.BillingMetricSecond,
		BillableValue: inferenceTimeSeconds,
		Metadata: map[string]interface{}{
			"modelid":         req.modelID,
			"memory_size_mb":  m.cfg.ModelService.EndpointConfig.MemorySizeInMB,
			"inference_count": count,
		},
	})
}
