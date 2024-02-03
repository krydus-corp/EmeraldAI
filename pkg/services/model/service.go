/*
 * File: service.go
 * Project: content
 * File Created: Friday, 19th March 2021 5:51:20 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package model

import (
	"github.com/pkg/errors"

	blob "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/blob"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	runtime "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/runtime"
	server "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/server"

	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
	api "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/api"
	config "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/config"

	batchWorker "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/worker/batch"
	endpointWorker "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/worker/endpoint"
	garbageWorker "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/worker/garbage"
	trainWorker "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/worker/train"
)

// Start starts all routines in the fetch service and waits for interrupt signal to gracefully shutdown.
func Start(cfg *config.Configuration) error {

	// Initialize Logger
	err := log.New(*cfg.Logger, log.InstanceZapLogger)
	if err != nil {
		return errors.Wrap(err, "could not instantiate logger")
	}

	// Initialize DB
	db, err := db.New(cfg.DB.URL, cfg.DB.Timeout)
	if err != nil {
		return err
	}
	defer db.Shutdown()

	// Initialize blob store
	blob, err := blob.NewBlob(cfg.BlobStore)
	if err != nil {
		return err
	}

	// Initialize platform
	plat := platform.NewPlatform()

	// Initialize workers
	trainWorker, err := trainWorker.New(cfg.ModelService.TrainJobQueueName, blob, db, plat, *cfg)
	if err != nil {
		return err
	}
	endpointWorker, err := endpointWorker.New(cfg.ModelService.EndpointJobQueueName, blob, db, plat, *cfg)
	if err != nil {
		return err
	}
	batchWorker, err := batchWorker.New(cfg.ModelService.BatchJobQueueName, blob, db, plat, *cfg)
	if err != nil {
		return err
	}
	garbageWorker, err := garbageWorker.New(cfg.ModelService.GarbageJobQueueName, blob, db, plat, *cfg)
	if err != nil {
		return err
	}

	trainWorker.Start()
	endpointWorker.Start()
	batchWorker.Start()
	garbageWorker.Start()

	// Initialize HTTP Server
	echoServer := server.New(server.Config{
		Port:           cfg.Server.Port,
		TimeoutSeconds: cfg.Server.Timeout,
		CrtFile:        cfg.Server.CrtFile,
		KeyFile:        cfg.Server.KeyFile,
		Debug:          cfg.Server.Debug,
	})
	v1 := echoServer.Group("/v1")

	modelSvc, err := api.Initialize(db, platform.NewPlatform(), cfg)
	if err != nil {
		return err
	}

	api.NewHTTP(modelSvc, v1)

	// Start Service
	echoServer.Start()

	log.Infof("model service initialized successfully")

	// Wait for os signal
	runtime.WaitForExit("model service exiting")

	return nil
}
