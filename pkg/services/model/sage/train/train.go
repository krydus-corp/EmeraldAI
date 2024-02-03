/*
 * File: spec.go
 * Project: train
 * File Created: Tuesday, 13th July 2021 2:48:52 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package train

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
	"github.com/aws/aws-sdk-go/aws"

	common "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	sage "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/sage"
)

var (
	ClassificationAttributes  = []string{"source-ref", "class"}
	ObjectDetectionAttributes = []string{"source-ref", "bounding-box"}

	// https://docs.amazonaws.cn/en_us/sagemaker/latest/dg/object-detection-api-config.html
	ObjectDetectionStaticHyperParameters = map[string]string{
		// --------------------------------------------------//
		// Static hyper-parameters //
		// --------------------------------------------------//
		"base_network":         "resnet-50",
		"use_pretrained_model": "1",
		"image_shape":          "512",
		"epochs":               "100",
		"lr_scheduler_step":    "10",
		"lr_scheduler_factor":  "0.8",
		"overlap_threshold":    "0.5",
		"nms_threshold":        "0.45",

		// --------------------------------------------------//
		// These are used in automatic hyper-parameter tuning //
		// --------------------------------------------------//
		// "optimizer":            "sgd",
		// "momentum":             "0.9",
		// "weight_decay":         "0.0005",
		// "mini_batch_size": 	   "32",
		// "learning_rate":        "0.001",

		//! --------------------------------------------------//
		//! These need to be updated dynamically 			  //
		//! --------------------------------------------------//
		"num_classes":          "",
		"num_training_samples": "",
		"label_width":          "",
	}

	// https://docs.amazonaws.cn/en_us/sagemaker/latest/dg/IC-Hyperparameter.html
	ClassificationStaticHyperParameters = map[string]string{
		// --------------------------------------------------//
		// Static hyper-parameters //
		// --------------------------------------------------//
		"num_layers":           "50",
		"use_pretrained_model": "1",
		"image_shape":          "3,224,224",
		"epochs":               "50",
		"lr_scheduler_step":    "10",
		"lr_scheduler_factor":  "0.8",
		"resize":               "224",
		"augmentation_type":    "crop_color_transform",
		"precision_dtype":      "float32",
		"multi_label":          "1",

		// --------------------------------------------------//
		// These are used in automatic hyper-parameter tuning //
		// --------------------------------------------------//
		// "optimizer":            "sgd",
		// "momentum":             "0.9",
		// "weight_decay":         "0.0001",
		// "mini_batch_size": 	   "32",
		// "learning_rate":        "0.001",

		//! --------------------------------------------------//
		//! These need to be updated dynamically 			  //
		//! --------------------------------------------------//
		"num_classes":          "",
		"num_training_samples": "",
	}

	// Rarely gain any efficiency gains frm going higher than this
	MaxBatchSize = 128.0
)

type Trainer struct {
	Config *Config
	Client *sagemaker.Client

	Input *sagemaker.CreateHyperParameterTuningJobInput
}

func New(config *Config, client *sagemaker.Client, algorithm models.ProjectAnnotationType) (*Trainer, error) {

	trainingJobName := fmt.Sprintf("train-job-%s", common.ShortUUID(17))
	s3TrainManifestKey := config.OutputDataPath + "/train.manifest"
	s3ValidationManifestKey := config.OutputDataPath + "/validation.manifest"

	var (
		attributeNames        []string
		staticHyperParameters map[string]string
		validationMetric      string
		baseImage             string
	)

	switch algorithm {
	case models.ProjectAnnotationTypeClassification:
		attributeNames = ClassificationAttributes
		staticHyperParameters = ClassificationStaticHyperParameters
		validationMetric = sage.Const_ClassificationObjectiveMetric
		baseImage = config.TrainImageClassification
	case models.ProjectAnnotationTypeBoundingBox:
		attributeNames = ObjectDetectionAttributes
		staticHyperParameters = ObjectDetectionStaticHyperParameters
		validationMetric = sage.Const_ObjectDetectionObjectiveMetric
		baseImage = config.TrainImageObjectDetection
	}

	trainJobInput := &sagemaker.CreateHyperParameterTuningJobInput{
		HyperParameterTuningJobName: aws.String(trainingJobName),
		TrainingJobDefinition: &types.HyperParameterTrainingJobDefinition{
			RoleArn: aws.String(config.ExecutionRoleArn),
			AlgorithmSpecification: &types.HyperParameterAlgorithmSpecification{
				TrainingImage:     aws.String(baseImage),
				TrainingInputMode: types.TrainingInputModePipe,
			},
			StoppingCondition: &types.StoppingCondition{
				MaxRuntimeInSeconds: config.Runtime.MaxRuntimeInSeconds,
			},
			RetryStrategy: &types.RetryStrategy{
				MaximumRetryAttempts: config.Runtime.MaximumRetryAttempts,
			},
			ResourceConfig: &types.ResourceConfig{
				InstanceCount:  config.Resource.InstanceCount,
				InstanceType:   config.Resource.InstanceType,
				VolumeSizeInGB: config.Resource.VolumeSizeInGB,
			},
			// Training data should be inside a subdirectory called "train"
			// Validation data should be inside a subdirectory called "validation"
			// The algorithm currently only supports fullyreplicated model (where data is copied onto each machine)
			InputDataConfig: []types.Channel{
				{
					ChannelName: aws.String("train"),
					DataSource: &types.DataSource{
						S3DataSource: &types.S3DataSource{
							S3DataType:             types.S3DataTypeAugmentedManifestFile,
							S3Uri:                  aws.String(s3TrainManifestKey),
							S3DataDistributionType: types.S3DataDistributionFullyReplicated,
							AttributeNames:         attributeNames,
						},
					},
					ContentType:       aws.String("application/x-recordio"),
					RecordWrapperType: types.RecordWrapperRecordio,
					CompressionType:   types.CompressionTypeNone,
				},
				{
					ChannelName: aws.String("validation"),
					DataSource: &types.DataSource{
						S3DataSource: &types.S3DataSource{
							S3DataType:             types.S3DataTypeAugmentedManifestFile,
							S3Uri:                  aws.String(s3ValidationManifestKey),
							S3DataDistributionType: types.S3DataDistributionFullyReplicated,
							AttributeNames:         attributeNames,
						},
					},
					ContentType:       aws.String("application/x-recordio"),
					RecordWrapperType: types.RecordWrapperRecordio,
					CompressionType:   types.CompressionTypeNone,
				},
			},
			OutputDataConfig: &types.OutputDataConfig{
				S3OutputPath: aws.String(config.OutputDataPath),
			},
			// Hyperparameter docs: https://docs.aws.amazon.com/sagemaker/latest/dg/IC-Hyperparameter.html
			StaticHyperParameters: staticHyperParameters,
		},
		HyperParameterTuningJobConfig: &types.HyperParameterTuningJobConfig{
			ResourceLimits: &types.ResourceLimits{
				MaxNumberOfTrainingJobs: common.Ptr(config.Runtime.MaxNumberOfTrainingJobs),
				// Restrict parallelism when using Baysian tuning strategy
				MaxParallelTrainingJobs: 1,
			},
			// https://docs.aws.amazon.com/sagemaker/latest/dg/automatic-model-tuning-how-it-works.html
			Strategy: types.HyperParameterTuningJobStrategyTypeBayesian,
			HyperParameterTuningJobObjective: &types.HyperParameterTuningJobObjective{
				MetricName: aws.String(validationMetric),
				Type:       types.HyperParameterTuningJobObjectiveTypeMaximize,
			},
			ParameterRanges: &types.ParameterRanges{
				CategoricalParameterRanges: []types.CategoricalParameterRange{
					{
						Name:   aws.String("optimizer"),
						Values: []string{"sgd", "adam", "rmsprop"},
					},
				},
				ContinuousParameterRanges: []types.ContinuousParameterRange{
					{
						Name:        aws.String("learning_rate"),
						MinValue:    aws.String("0.0001"),
						MaxValue:    aws.String("0.01"),
						ScalingType: types.HyperParameterScalingTypeAuto,
					},
					{
						Name:        aws.String("weight_decay"),
						MinValue:    aws.String("0.0001"),
						MaxValue:    aws.String("0.1"),
						ScalingType: types.HyperParameterScalingTypeAuto,
					},
					{
						Name:        aws.String("momentum"),
						MinValue:    aws.String("0.0001"),
						MaxValue:    aws.String("0.1"),
						ScalingType: types.HyperParameterScalingTypeAuto,
					},
				},
			},
			TrainingJobEarlyStoppingType: types.TrainingJobEarlyStoppingTypeAuto,
		},
	}

	return &Trainer{
		Config: config,
		Client: client,
		Input:  trainJobInput,
	}, nil
}

func (t *Trainer) Train(numClasses, numTrainingSamples, numValidationSamples, forcePaddingLabelWidth int32) error {
	// Dynamically set hyperparameters
	t.Input.TrainingJobDefinition.StaticHyperParameters["num_classes"] = fmt.Sprint(numClasses)
	t.Input.TrainingJobDefinition.StaticHyperParameters["num_training_samples"] = fmt.Sprint(numTrainingSamples)
	if strings.Contains(*t.Input.TrainingJobDefinition.AlgorithmSpecification.TrainingImage, "object-detection") {
		log.Debugf("updating 'label_width' with value: %d", forcePaddingLabelWidth)
		t.Input.TrainingJobDefinition.StaticHyperParameters["label_width"] = fmt.Sprint(forcePaddingLabelWidth)
	}

	maxBatchSize := math.Min(float64(numTrainingSamples), float64(numValidationSamples))
	maxBatchSize = math.Min(maxBatchSize, MaxBatchSize)
	t.Input.HyperParameterTuningJobConfig.ParameterRanges.IntegerParameterRanges = []types.IntegerParameterRange{
		{
			Name:        aws.String("mini_batch_size"),
			MinValue:    aws.String(fmt.Sprint(math.Min(maxBatchSize, 8) - 1)),
			MaxValue:    aws.String(fmt.Sprint(maxBatchSize)),
			ScalingType: types.HyperParameterScalingTypeAuto,
		},
	}

	if _, err := t.Client.CreateHyperParameterTuningJob(context.TODO(), t.Input); err != nil {
		return err
	}
	return nil
}

func (t *Trainer) Metrics(trainingJobName string) (metrics map[string]interface{}, err error) {
	output, err := t.Client.DescribeTrainingJob(context.TODO(), &sagemaker.DescribeTrainingJobInput{
		TrainingJobName: &trainingJobName,
	})
	if err != nil {
		return nil, err
	}

	metrics = make(map[string]interface{})
	for _, metric := range output.FinalMetricDataList {
		metrics[*metric.MetricName] = metric.Value
	}

	// Billing related metrics
	metrics["BillableTimeInSeconds"] = *output.BillableTimeInSeconds * output.ResourceConfig.InstanceCount

	return metrics, nil
}
