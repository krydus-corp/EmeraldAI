/*
 * File: config.go
 * Project: content
 * File Created: Friday, 19th March 2021 5:53:07 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package config

import (
	blob "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/blob"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
	endpoint "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/sage/endpoint"
	garbage "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/sage/garbage"
	train "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/sage/train"

	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	worker "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/worker"
)

type Configuration struct {
	Logger       *log.Configuration      `yaml:"log,omitempty"`
	DB           *platform.Configuration `yaml:"database,omitempty"`
	BlobStore    *blob.Configuration     `yaml:"blob_store,omitempty"`
	Server       *Server                 `yaml:"server,omitempty"`
	ModelService *ModelService           `yaml:"model_service,omitempty"`
}
type ModelService struct {
	TrainJobQueueName    string `yaml:"train_job_queue_name,omitempty"`
	EndpointJobQueueName string `yaml:"endpoint_job_queue_name,omitempty"`
	BatchJobQueueName    string `yaml:"batch_job_queue_name,omitempty"`
	GarbageJobQueueName  string `yaml:"garbage_job_queue_name,omitempty"`

	TrainWorkerConfig    worker.Config `yaml:"train_worker_config,omitempty"`
	EndpointWorkerConfig worker.Config `yaml:"endpoint_worker_config,omitempty"`
	BatchWorkerConfig    worker.Config `yaml:"batch_worker_config,omitempty"`
	GarbageWorkerConfig  worker.Config `yaml:"garbage_worker_config,omitempty"`

	TrainConfig    train.Config    `yaml:"train_config,omitempty"`
	EndpointConfig endpoint.Config `yaml:"endpoint_config,omitempty"`
	GarbageConfig  garbage.Config  `yaml:"garbage_config,omitempty"`
}

// Server holds data necessary for server configuration
type Server struct {
	Port    string `yaml:"port,omitempty"`
	Debug   bool   `yaml:"debug,omitempty"`
	Timeout int    `yaml:"timeout_seconds,omitempty"`
	CrtFile string `yaml:"crt_file"`
	KeyFile string `yaml:"key_file"`
}
