/*
 * File: config.go
 * Project: train
 * File Created: Tuesday, 16th August 2022 12:55:23 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package train

import (
	"github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
)

type Config struct {
	ExecutionRoleArn          string `yaml:"execution_role_arn,omitempty"`
	TrainImageClassification  string `yaml:"train_image_classification,omitempty"`
	TrainImageObjectDetection string `yaml:"train_image_object_detection,omitempty"`

	Runtime struct {
		MaxRuntimeInSeconds     int32 `yaml:"max_runtime_seconds,omitempty"`
		MaximumRetryAttempts    int32 `yaml:"maximum_retry_attempts,omitempty"`
		MaxNumberOfTrainingJobs int32 `yaml:"max_number_training_jobs,omitempty"`
	} `yaml:"runtime,omitempty"`

	Resource struct {
		InstanceCount  int32                      `yaml:"instance_count,omitempty"`
		InstanceType   types.TrainingInstanceType `yaml:"instance_type,omitempty"`
		VolumeSizeInGB int32                      `yaml:"volume_size_gb,omitempty"`
	} `yaml:"resource,omitempty"`

	SendGrid struct {
		EmailSource                string `yaml:"email_source"`
		EmailSubject               string `yaml:"email_subject"`
		TrainingCompleteTemplateID string `yaml:"training_complete_template_id"`
		TrainingSuccessMessage     string `yaml:"training_success_message"`
		TrainingFailedMessage      string `yaml:"training_failed_message"`
	}

	// Dynamically set
	OutputDataPath         string `yaml:"-"`
	NumClasses             int32  `yaml:"-"`
	NumTrainingSamples     int32  `yaml:"-"`
	NumValidationSamples   int32  `yaml:"-"`
	ForcePaddingLabelWidth int32  `yaml:"-"`
}
