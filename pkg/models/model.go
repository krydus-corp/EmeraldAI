/*
 * File: model.go
 * Project: models
 * File Created: Friday, 7th May 2021 4:38:45 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package models

import (
	"net/url"
	"path"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ModelState int

const (
	ModelStateUnknown ModelState = iota
	ModelStateInitialized
	ModelStateTrained
	ModelStateTraining
	ModelStateErr
)

func (s ModelState) String() string {
	return [...]string{"UNKNOWN", "INITIALIZED", "TRAINED", "TRAINING", "ERR"}[s]
}

// Model represents model domain model
//
// swagger:model Model
type Model struct {
	// ID of the Model
	//
	// swagger:strfmt bsonobjectid
	ID primitive.ObjectID `json:"id" bson:"_id"`
	// Name associated with Model
	//
	Name string `json:"name" bson:"name"`
	// Training job name - internal name associated with Sagemaker job
	//
	TrainingJobName string `json:"-" bson:"training_job_name"`
	// UserID associated with Model
	//
	UserID string `json:"userid" bson:"userid"`
	// ProjectID associated with Model
	//
	ProjectID string `json:"projectid" bson:"projectid"`
	// DatasetID associated with Model
	//
	DatasetID string `json:"datasetid" bson:"datasetid"`
	// Model filepath
	//
	Path string `json:"-" bson:"path"`
	// Model metrics
	//
	Metrics map[string]interface{} `json:"metrics" bson:"metrics"`
	// State of the current model
	//
	State string `json:"state" bson:"state"`
	// Last error (if any) associated with the model
	//
	LastError *string `json:"error" bson:"error"`
	// Mapping of label to integer representation
	//
	IntegerMapping map[string]int `json:"-" bson:"integer_mapping"`
	// Optional Train Parameters
	//
	Parameters TrainParameters `json:"parameters" bson:"parameters"`
	// Preprocessing
	//
	Preprocessing Preprocessors `json:"preprocessing" bson:"preprocessing"`
	// Augmentation
	//
	Augmentation Augmentations `json:"augmentation" bson:"augmentation"`
	// Deployment Info
	//
	Deployment Deployment `json:"deployment" bson:"deployment"`
	// Batch Info
	//
	Batch Batch `json:"batch" bson:"batch"`
	// Internal metadata
	//
	Metadata map[string]interface{} `json:"-" bson:"metadata"`

	TrainStartedAt time.Time `json:"train_started_at" bson:"train_started_at"`
	TrainEndedAt   time.Time `json:"train_ended_at" bson:"train_ended_at"`
	ErrorAt        time.Time `json:"error_at" bson:"error_at"`
	CreatedAt      time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" bson:"updated_at"`
}

// Model represents Parameters domain model
//
// swagger:model Parameters
type TrainParameters struct {
	// Runtime parameters
	Runtime struct {
		// Specifies a limit to how long a model hyperparameter training job can run.
		MaxRuntimeInSeconds *int32 `json:"max_runtime_seconds,omitempty"`
		// The number of times to retry the job if it fails due to an internal server error
		MaximumRetryAttempts *int32 `json:"maximum_retry_attempts,omitempty"`
		// Max number of train attempts.
		MaxNumberOfTrainingJobs *int32 `json:"max_number_training_jobs,omitempty"`
	} `json:"runtime,omitempty"`

	// Resource parameters
	Resource struct {
		// AWS instance type to train on.
		InstanceType *string `json:"instance_type,omitempty"`
	} `json:"resource,omitempty"`

	// =========================================
	// Tunable classification hyper-parameters
	// =========================================
	//
	// Object Detection Parameters:
	// "epochs":               	<int>						// The number of training epochs.
	// "lr_scheduler_step":    	<int>						// The epochs at which to reduce the learning rate. The learning rate is reduced by lr_scheduler_factor at epochs listed in a comma-delimited string: "epoch1, epoch2, ...". For example, if the value is set to "10, 20" and the lr_scheduler_factor is set to 1/2, then the learning rate is halved after 10th epoch and then halved again after 20th epoch.
	// "lr_scheduler_factor":  	<float(0:1)>				// The ratio to reduce learning rate. Used in conjunction with the lr_scheduler_step parameter defined as lr_new = lr_old * lr_scheduler_factor.
	// "overlap_threshold":    	<float(0:1]> 				// The evaluation overlap threshold.
	// "nms_threshold":       	<float(0:1]> 				// The non-maximum suppression threshold.
	// "learning_rate":  		<float(0:1]>,<float(0:1]>	// The initial learning rate.
	// "weight_decay":			<float(0:1)>,<float(0:1)>	// The weight decay coefficient for sgd and rmsprop. Ignored for other optimizers.
	// =========================================
	// Classification Parameters:
	// =========================================
	// "num_layers":           <int>,			// Number of layers for the network. For data with large image size (for example, 224x224 - like ImageNet), we suggest selecting the number of layers from the set [18, 34, 50, 101, 152, 200]. For data with small image size (for example, 28x28 - like CIFAR), we suggest selecting the number of layers from the set [20, 32, 44, 56, 110].
	// "epochs":               	<int>			// The number of training epochs.
	// "lr_scheduler_step":    	<int>			// The epochs at which to reduce the learning rate. The learning rate is reduced by lr_scheduler_factor at epochs listed in a comma-delimited string: "epoch1, epoch2, ...". For example, if the value is set to "10, 20" and the lr_scheduler_factor is set to 1/2, then the learning rate is halved after 10th epoch and then halved again after 20th epoch.
	// "lr_scheduler_factor":  	<float(0:1)>	// The ratio to reduce learning rate. Used in conjunction with the lr_scheduler_step parameter defined as lr_new = lr_old * lr_scheduler_factor.
	HyperParameters map[string]interface{} `json:"hyperparameters,omitempty"`
}

func NewModel(name, userid, projectid, datasetid, s3bucket string, preprocessors Preprocessors, augmentations Augmentations, metadata ...map[string]interface{}) Model {
	id := primitive.NewObjectID()
	meta := make(map[string]interface{})
	if len(metadata) > 0 {
		meta = metadata[0]
	}

	return Model{
		ID:        id,
		Name:      name,
		UserID:    userid,
		ProjectID: projectid,
		DatasetID: datasetid,
		Path:      "s3://" + path.Join(s3bucket, userid, "models", id.Hex()),
		Metrics:   make(map[string]interface{}),
		State:     ModelStateInitialized.String(),

		TrainStartedAt: time.Time{},
		TrainEndedAt:   time.Time{},
		ErrorAt:        time.Time{},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),

		IntegerMapping: make(map[string]int),

		Parameters:    TrainParameters{},
		Preprocessing: preprocessors,
		Augmentation:  augmentations,

		Deployment: Deployment{},
		Batch:      Batch{},

		Metadata: meta,
	}
}

func (m *Model) Key() string {
	u, _ := url.Parse(m.Path)
	return strings.TrimPrefix(u.Path, "/")
}
