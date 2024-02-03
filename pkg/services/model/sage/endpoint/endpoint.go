/*
 * File: endpoint.go
 * Project: endpoint
 * File Created: Tuesday, 16th August 2022 2:27:43 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package endpoint

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
	"github.com/aws/aws-sdk-go/aws"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
)

const (
	sleepTime = 300 * time.Microsecond
)

type Endpoint struct {
	Config *Config
	Client *sagemaker.Client

	trainingImage    *string
	s3ModelArtifacts *string

	modelName          string
	endpointConfigName string
	endpointName       string
}

func (e *Endpoint) Describe() (endpointName string, endpointConfigName string, modelName string) {
	return e.endpointName, e.endpointConfigName, e.modelName
}

func New(config *Config, client *sagemaker.Client, trainingJobName string) (*Endpoint, error) {
	trainingJobOutput, err := client.DescribeTrainingJob(context.TODO(), &sagemaker.DescribeTrainingJobInput{
		TrainingJobName: &trainingJobName,
	})
	if err != nil {
		return nil, err
	}

	return &Endpoint{
		Config:             config,
		Client:             client,
		trainingImage:      trainingJobOutput.AlgorithmSpecification.TrainingImage,
		s3ModelArtifacts:   trainingJobOutput.ModelArtifacts.S3ModelArtifacts,
		modelName:          fmt.Sprintf("model-%s", common.ShortUUID(17)),
		endpointConfigName: fmt.Sprintf("endpoint-cfg-%s", common.ShortUUID(17)),
		endpointName:       fmt.Sprintf("endpoint-%s", common.ShortUUID(17)),
	}, nil
}

func (e *Endpoint) CreateEndpoint() error {
	// Create model deployment
	// https://sagemaker-examples.readthedocs.io/en/latest/introduction_to_amazon_algorithms/imageclassification_caltech/Image-classification-fulltraining.html
	if _, err := e.Client.CreateModel(context.TODO(), &sagemaker.CreateModelInput{
		Tags:             []types.Tag{{Key: aws.String("environment"), Value: aws.String(e.Config.ResourceEnv)}},
		ModelName:        aws.String(e.modelName),
		ExecutionRoleArn: aws.String(e.Config.ExecutionRoleArn),
		PrimaryContainer: &types.ContainerDefinition{
			Image:        e.trainingImage,
			ModelDataUrl: e.s3ModelArtifacts,
		},
	}); err != nil {
		return err
	}

	// Create endpoint configuration
	// https://docs.aws.amazon.com/sagemaker/latest/dg/serverless-endpoints-create.html
	if _, err := e.Client.CreateEndpointConfig(context.TODO(), &sagemaker.CreateEndpointConfigInput{
		Tags:               []types.Tag{{Key: aws.String("environment"), Value: aws.String(e.Config.ResourceEnv)}},
		EndpointConfigName: aws.String(e.endpointConfigName),
		ProductionVariants: []types.ProductionVariant{
			{
				ModelName:   aws.String(e.modelName),
				VariantName: aws.String("AllTraffic"),
				ServerlessConfig: &types.ProductionVariantServerlessConfig{
					MemorySizeInMB: aws.Int32(e.Config.MemorySizeInMB),
					MaxConcurrency: aws.Int32(e.Config.MaxConcurrency),
				},
			},
		},
	}); err != nil {
		return err
	}

	// Create endpoint - this may take a few minutes
	if _, err := e.Client.CreateEndpoint(context.TODO(), &sagemaker.CreateEndpointInput{
		Tags:               []types.Tag{{Key: aws.String("environment"), Value: aws.String(e.Config.ResourceEnv)}},
		EndpointConfigName: aws.String(e.endpointConfigName),
		EndpointName:       aws.String(e.endpointName),
	}); err != nil {
		return err
	}

	return nil
}

func (e *Endpoint) DeleteEndpoint(endpointName string) error {

	// Delete endpoint - this may take a few minutes
	if _, err := e.Client.DeleteEndpoint(context.TODO(), &sagemaker.DeleteEndpointInput{
		EndpointName: aws.String(endpointName),
	}); err != nil {
		return err
	}
	// Avoid rate limiter.
	time.Sleep(sleepTime)

	// Delete Endpont Config
	endpointDescription, err := e.Client.DescribeEndpoint(context.TODO(), &sagemaker.DescribeEndpointInput{
		EndpointName: aws.String(endpointName),
	})
	if err != nil {
		return err
	}
	// Avoid rate limiter.
	time.Sleep(sleepTime)

	if _, err := e.Client.DeleteEndpointConfig(context.TODO(), &sagemaker.DeleteEndpointConfigInput{
		EndpointConfigName: endpointDescription.EndpointConfigName,
	}); err != nil {
		return err
	}

	return nil
}
