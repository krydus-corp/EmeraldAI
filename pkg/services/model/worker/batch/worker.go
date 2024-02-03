/*
 * File: worker.go
 * Project: batch
 * File Created: Sunday, 11th September 2022 3:47:56 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package batch

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sagemakerruntime"

	blob "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/blob"
	common "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	sqs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/aws/sqs"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	thumbnail "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/image"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	worker "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/worker"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
	modelConfig "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/config"
	sage "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/sage"
)

const (
	// BatchInsertSize is the number of documents we send to the DB in a insertMany operation
	BatchInsertSize = 1000
)

type WorkerPool struct {
	Platform *platform.Platform
	DB       *db.DB
	Blob     *blob.Blob
	Config   modelConfig.Configuration

	sagemakerRuntimeClient *sagemakerruntime.Client
	consumer               sqs.Consumer
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
		WorkerPool:        cfg.ModelService.BatchWorkerConfig.Concurrency,
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

		sagemakerRuntimeClient: sagemakerruntime.NewFromConfig(awsConfig),
		consumer:               consumer,
	}, nil
}

func (w *WorkerPool) Start() {
	// Start subscriber
	log.Infof("starting subscriber loop on queue=%s", w.Config.ModelService.BatchJobQueueName)
	go w.consumer.Consume(w.callback)
}

// callback is a method for handling a batch Event
// Any errors propagated up from this method will requeue the message for processed
func (w *WorkerPool) callback(msg *sqs.Message) error {
	// Decode the subscriber event
	var event Event
	if err := msg.Decode(&event); err != nil {
		log.Errorf("unable to decode batch event; msg=%s error=%s", string(msg.Body()), err.Error())
		return nil
	}
	log.Debugf("Received message=%s", event.ToJSON())

	// Retrieve associated model
	model, err := w.Platform.ModelDB.View(w.DB, event.UserID, event.ModelID)
	if err != nil {
		log.Errorf("unable to view model associated with batch event; model=%s error=%s", event.ModelID, err.Error())
		return nil
	}

	if model.Deployment.EndpointName == "" || model.Deployment.Status != models.DeploymentStatusInService.String() {
		log.Errorf("model is not deployed; model=%s", event.ModelID, err.Error())
		return w.updateLastErr(errors.New("model not deployed"), &model)
	}

	if model.Batch.Status == models.BatchStatusRunning.String() {
		return nil
	}

	// Clear all previous predictions
	if err := w.Platform.PredictionDB.DeleteMany(w.DB, model.UserID, model.ID.Hex()); err != nil {
		log.Errorf("unable to clear previous predictions for batch event; model=%s error=%s", event.ModelID, err.Error())
		return w.updateLastErr(err, &model)
	}

	// Update batch status
	model.Batch.LastError = nil
	model.Batch.Status = models.BatchStatusRunning.String()
	model.Batch.Threshold = sage.Const_BatchPredictionConfidenceThreshold
	if err := w.Platform.ModelDB.Update(w.DB, models.Model{
		ID:     model.ID,
		UserID: model.UserID,
		Batch:  model.Batch,
	}); err != nil {
		log.Errorf("error updating batch apply job at DB; err=%s", err.Error())
		return w.updateLastErr(err, &model)
	}

	// Spin up a worker pool for running content through realtime
	log.Debugf("Starting worker pool...")
	config := worker.Config{
		Concurrency:      int(w.Config.ModelService.EndpointConfig.MaxConcurrency),
		RetryAttempts:    w.Config.ModelService.BatchWorkerConfig.RetryAttempts,
		RetryWaitSeconds: w.Config.ModelService.BatchWorkerConfig.RetryWaitSeconds,
		RetryBackoff:     w.Config.ModelService.BatchWorkerConfig.RetryBackoff,
		Name:             fmt.Sprintf("batch-job-model-%s", model.ID.Hex()),
	}

	pool := worker.New[models.Prediction](config).Start()

	log.Debugf("Starting pool receiver")
	// Update DB with predictions
	batch, errs := []models.Prediction{}, []error{}
	sent, received, batches, totalContent := 0, 0, 0, 0
	go func(totalContent *int) {
		for result := range pool.OutChan {
			received++

			if result.Err != nil {
				if errors.Is(result.Err, ErrEmptyPredictions) {
					errs = append(errs, ErrEmptyPredictions)
					continue
				}

				log.Errorf("unexpected prediction error for batch event; model=%s content=%s error=%s", event.ModelID, result.Value.ContentID, result.Err.Error())
				errs = append(errs, result.Err)
				continue
			}

			batch = append(batch, result.Value)

			// Update prediction at DB
			if len(batch)%BatchInsertSize == 0 {
				log.Debugf("Processing batch %d", batches)

				if _, err := w.Platform.PredictionDB.Create(w.DB, batch); err != nil {
					log.Errorf("unable to create predictions for batch event; model=%s content=%s error=%s", event.ModelID, result.Value.ContentID, err.Error())
					errs = append(errs, err)
				}

				model.Batch.TotalContent = *totalContent
				model.Batch.CompletedContent += len(batch)
				if err := w.Platform.ModelDB.Update(w.DB, models.Model{
					ID:     model.ID,
					UserID: model.UserID,
					Batch:  model.Batch,
				}); err != nil {
					log.Errorf("error updating batch apply job at DB; err=%s", err.Error())
					continue
				}

				batches += 1
				batch = nil
			}
		}

		// Final batch
		log.Debugf("Processing final batch")
		if len(batch) > 0 {
			if _, err := w.Platform.PredictionDB.Create(w.DB, batch); err != nil {
				log.Errorf("unable to create predictions for batch event; model=%s error=%s", event.ModelID, err.Error())
				errs = append(errs, err)
			}

			model.Batch.TotalContent = *totalContent
			model.Batch.CompletedContent += len(batch)
			if err := w.Platform.ModelDB.Update(w.DB, models.Model{
				ID:     model.ID,
				UserID: model.UserID,
				Batch:  model.Batch,
			}); err != nil {
				log.Errorf("error updating batch apply job at DB; err=%s", err.Error())
			}

			batches += 1
			received += len(batch)
			batch = nil
		}
	}(&totalContent)

	// Curse through annotations and feed to worker pool
	log.Debugf("Starting feed loop")
	page := 0

	for {
		p := models.PaginationReq{Limit: 1000, Page: page}.Transform()
		content, total, err := w.Platform.ContentDB.FindAnnotated(w.DB, model.UserID, model.ProjectID, "", "or", p)
		if err != nil {
			errs = append(errs, err)
			break
		}

		if len(content) == 0 {
			break
		}

		thumbnailsize, err := strconv.ParseInt(event.ThumbnailSize, 0, 64)
		if err != nil {
			thumbnailsize = thumbnail.BoundingBoxDefaultThumbnailSize
		}
		for _, c := range content {
			args := inferArgs{
				content:                c,
				model:                  model,
				blob:                   w.Blob,
				sagemakerRuntimeClient: w.sagemakerRuntimeClient,
				thumbnailSize:          int(thumbnailsize),
			}
			pool.InChan <- args.infer
			sent++
		}

		page++
		totalContent = total
	}

	// Wait for completion
	for received != sent {
		log.Debugf("Waiting for batch to finish; sent=%d received=%d model=%s", sent, received, model.ID.Hex())
		time.Sleep(1 * time.Second)
	}

	pool.Stop()

	if len(errs) > 0 {
		err := common.CombineErrors(errs).Error()
		log.Errorf("one or more errors occurred during batch apply; errs=%s", err)
		model.Batch.Status = models.BatchStatusCompleteWithErr.String()
		model.Batch.EndedAt = time.Now()
		model.Batch.LastError = &err

	} else {
		model.Batch.Status = models.BatchStatusComplete.String()
		model.Batch.EndedAt = time.Now()
		model.Batch.LastError = nil
	}

	if err := w.Platform.ModelDB.Update(w.DB, models.Model{
		ID:     model.ID,
		UserID: model.UserID,
		Batch:  model.Batch,
	}); err != nil {
		log.Errorf("error updating batch apply job at DB; err=%s", err.Error())
		return err
	}

	return nil
}

func (w *WorkerPool) updateLastErr(err error, model *models.Model) error {
	if err != nil {
		if err := w.Platform.ModelDB.Update(w.DB, models.Model{
			ID:     model.ID,
			UserID: model.UserID,
			Batch: models.Batch{
				Status:    models.BatchStatusErr.String(),
				StartedAt: model.Batch.StartedAt,
				EndedAt:   time.Now(),
				LastError: common.Ptr(err.Error()),
			},
		}); err != nil {
			log.Errorf("error updating DB with failed batch job; modelid=%s err=%s", model.ID.Hex(), err.Error())
		}
	}
	return err
}
