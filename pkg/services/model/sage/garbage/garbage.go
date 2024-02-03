package garbage

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"

	types "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

const (
	sleepTime = 300 * time.Microsecond
)

type Garbage struct {
	Config           *Config
	SagemakerClient  *sagemaker.Client
	CloudwatchClient *cloudwatch.Client
	DB               *db.DB
	Platform         *platform.Platform

	cacheENV map[string]string // arn to tag environment
}

func New(config *Config, sagemakerClient *sagemaker.Client, cloudwatchClient *cloudwatch.Client, db *db.DB, platform *platform.Platform) *Garbage {
	return &Garbage{
		Config:           config,
		SagemakerClient:  sagemakerClient,
		CloudwatchClient: cloudwatchClient,
		DB:               db,
		Platform:         platform,

		cacheENV: make(map[string]string),
	}
}

func (g *Garbage) CleanEndpoints() error {
	t := time.Now().AddDate(0, 0, g.Config.RemoveAfterDaysUnused*-1)

	// Check Endpoints
	var nextToken *string = nil
	for {
		// Get next set of endpoints
		endpoints, err := g.SagemakerClient.ListEndpoints(
			context.TODO(),
			&sagemaker.ListEndpointsInput{
				LastModifiedTimeBefore: &t,
				NextToken:              nextToken,
			},
		)
		if err != nil {
			log.Errorf("error getting list of endpoints")
			return err
		}
		// Avoid rate limiter.
		time.Sleep(sleepTime)
		nextToken = endpoints.NextToken

		// Check endpoints for possible cleanup.
		for _, endpoint := range endpoints.Endpoints {
			endpointDescription, err := g.SagemakerClient.DescribeEndpoint(
				context.TODO(),
				&sagemaker.DescribeEndpointInput{
					EndpointName: endpoint.EndpointName,
				},
			)
			if err != nil {
				log.Errorf("error getting endpoint description endpoint=%s", *endpoint.EndpointName)
				return err
			}
			// Avoid rate limiter.
			time.Sleep(sleepTime)

			// Check if resource tag is this env
			foundEnvTag := false

			// Check if arn is in cache or else call list tags
			if val, ok := g.cacheENV[*endpointDescription.EndpointArn]; ok {
				if val == "environment" {
					foundEnvTag = true
				}
			} else {
				tags, err := g.SagemakerClient.ListTags(
					context.TODO(),
					&sagemaker.ListTagsInput{
						ResourceArn: endpointDescription.EndpointArn,
					},
				)

				if err != nil {
					log.Errorf("error getting list of endpoint=%s tags", *endpoint.EndpointName)
				}

				// Avoid rate limiter.
				time.Sleep(sleepTime)

				// Check for tag ENV to see if it matches this environment.
				for _, tag := range tags.Tags {
					if *tag.Key == "environment" {
						if *tag.Value == g.Config.ResourceEnv {
							foundEnvTag = true
						}
						// Cache resource environment
						g.cacheENV[*endpointDescription.EndpointArn] = *tag.Value
						break
					}
				}
			}

			// Resource is not for this environment. Continue to next endpoint.
			if !(foundEnvTag) {
				continue
			}

			// Check if endpoint exists in db else delete.
			count, err := g.checkEndpointExists(*endpoint.EndpointName)
			if err != nil {
				log.Errorf("error checking if endpoint exists in db endpoint=%s", *endpoint.EndpointName)
				return err
			}
			// Avoid rate limiter.
			time.Sleep(sleepTime)

			if count == 0 {
				if err := g.deleteEndpointServices(endpoint.EndpointName, endpointDescription.EndpointConfigName); err != nil {
					return err
				}
				// Avoid rate limiter.
				time.Sleep(sleepTime)
				continue
			}

			// Check if endpoint is failed or out of service.
			if endpointDescription.EndpointStatus == "OutOfService" ||
				endpointDescription.EndpointStatus == "Failed" {
				if err := g.deleteEndpointServices(endpoint.EndpointName, endpointDescription.EndpointConfigName); err != nil {
					return err
				}
				// Avoid rate limiter.
				time.Sleep(sleepTime)
				continue
			}

			// Check if invocations have been made.
			invocations, err := g.getEndpointInvocationCount(*endpoint.EndpointName)
			if err != nil {
				return err
			}
			// Avoid rate limiter.
			time.Sleep(sleepTime)
			if invocations == 0 {
				if err := g.deleteEndpointServices(endpoint.EndpointName, endpointDescription.EndpointConfigName); err != nil {
					return err
				}
				// Avoid rate limiter.
				time.Sleep(sleepTime)
			}
		}

		// Stop if not next set of endpoints.
		if endpoints.NextToken == nil {
			break
		}
	}

	return nil
}

func (g *Garbage) CleanModels() error {
	t := time.Now().AddDate(0, 0, g.Config.RemoveAfterDaysUnused*-1)
	var nextToken *string = nil
	for {
		models, err := g.SagemakerClient.ListModels(
			context.TODO(),
			&sagemaker.ListModelsInput{
				CreationTimeBefore: &t,
				NextToken:          nextToken,
			},
		)
		if err != nil {
			return err
		}
		// Avoid rate limiter.
		time.Sleep(sleepTime)

		nextToken = models.NextToken

		for _, model := range models.Models {
			// Check if resource tag is this environment.
			foundEnvTag := false

			// Check if arn is in cache else call list tags.
			if val, ok := g.cacheENV[*model.ModelArn]; ok {
				if val == "environment" {
					foundEnvTag = true
				}
			} else {
				tags, err := g.SagemakerClient.ListTags(
					context.TODO(),
					&sagemaker.ListTagsInput{
						ResourceArn: model.ModelArn,
					},
				)

				if err != nil {
					log.Errorf("error getting list of model=%s tags", *model.ModelName)
					continue
				}

				// Avoid rate limiter.
				time.Sleep(sleepTime)

				// Check for tag ENV to see if it matches this environment.
				for _, tag := range tags.Tags {
					if *tag.Key == "environment" {
						if *tag.Value == g.Config.ResourceEnv {
							foundEnvTag = true
						}
						g.cacheENV[*model.ModelArn] = *tag.Value
						break
					}
				}
			}
			// Resource is not for this environment. Continue to next model.
			if !(foundEnvTag) {
				continue
			}

			// Check if model exits in database else delete
			count, err := g.checkModelExists(*model.ModelName)
			if err != nil {
				return err
			}
			// Avoid rate limiter.
			time.Sleep(sleepTime)

			if count == 0 {
				if _, err := g.SagemakerClient.DeleteModel(
					context.TODO(),
					&sagemaker.DeleteModelInput{
						ModelName: model.ModelName,
					},
				); err != nil {
					return err
				}
				log.Infof("Model=%s deleted..", *model.ModelName)
				// Avoid rate limiter.
				time.Sleep(sleepTime)
			}
		}

		if models.NextToken == nil {
			break
		}
	}
	return nil
}

func (g *Garbage) getEndpointInvocationCount(endpoint string) (float64, error) {
	dimension := []types.Dimension{
		{Name: aws.String("EndpointName"), Value: aws.String(endpoint)},
		{Name: aws.String("VariantName"), Value: aws.String("AllTraffic")},
	}
	start := time.Now().AddDate(0, 0, g.Config.RemoveAfterDaysUnused*-1)
	endtime := time.Now()
	var period int32 = int32(86400 * g.Config.RemoveAfterDaysUnused)
	stats, err := g.CloudwatchClient.GetMetricStatistics(context.TODO(),
		&cloudwatch.GetMetricStatisticsInput{
			MetricName: aws.String("Invocations"),
			Namespace:  aws.String("SageMaker"),
			Dimensions: dimension,
			Period:     aws.Int32(period),
			StartTime:  &start,
			EndTime:    &endtime,
			Statistics: []types.Statistic{types.StatisticSum},
		},
	)
	if err != nil {
		return 0, err
	}

	if len(stats.Datapoints) > 0 {
		var invocationMax float64 = 0
		for _, point := range stats.Datapoints {
			if *point.Sum > invocationMax {
				invocationMax = *point.Sum
			}
		}
		return invocationMax, nil
	}

	return 0, nil
}

func (g *Garbage) checkEndpointExists(endpointName string) (int64, error) {
	return g.Platform.ModelDB.AWSEndpointExists(g.DB, endpointName)
}

func (g *Garbage) checkModelExists(modelName string) (int64, error) {
	return g.Platform.ModelDB.AWSModelExists(g.DB, modelName)
}

func (g *Garbage) updateModelEndpointStatus(endpointName string) error {
	return g.Platform.ModelDB.UpdateEndpointStatus(g.DB, endpointName)
}

func (g *Garbage) deleteEndpointServices(endpointName, endpointConfigName *string) error {
	if _, err := g.SagemakerClient.DeleteEndpointConfig(
		context.TODO(),
		&sagemaker.DeleteEndpointConfigInput{
			EndpointConfigName: endpointConfigName,
		},
	); err != nil {
		log.Errorf("error deleting endpoint config endopint_config_name=%s", *endpointConfigName)
		return err
	}
	log.Infof("endpoint config deleted endopint_config_name=%s", *endpointConfigName)
	// Avoid rate limiter.
	time.Sleep(sleepTime)

	if _, err := g.SagemakerClient.DeleteEndpoint(
		context.TODO(),
		&sagemaker.DeleteEndpointInput{
			EndpointName: endpointName,
		},
	); err != nil {
		log.Errorf("error deleting endpoint=%s", *endpointName)
		return err
	}
	log.Infof("endpoint deleted endpoint=%s", *endpointName)

	err := g.updateModelEndpointStatus(*endpointName)
	if err != nil {
		log.Errorf("error updating model endpoint status endpoint=%s", endpointName)
		return err
	}
	log.Infof("model endpoint status updated endpoint=%s", endpointName)
	// Avoid rate limiter.
	time.Sleep(sleepTime)

	return nil
}

func (g *Garbage) Delete(modelName, endpointName string) error {
	// Delete endpoint - this may take a few minutes
	if _, err := g.SagemakerClient.DeleteEndpoint(context.TODO(), &sagemaker.DeleteEndpointInput{
		EndpointName: aws.String(endpointName),
	}); err != nil {
		return err
	}
	log.Infof("Deleted endpoint=%s", endpointName)

	// Delete Endpont Config
	endpointDescription, err := g.SagemakerClient.DescribeEndpoint(context.TODO(), &sagemaker.DescribeEndpointInput{
		EndpointName: aws.String(endpointName),
	})
	if err != nil {
		return err
	}

	if _, err := g.SagemakerClient.DeleteEndpointConfig(context.TODO(), &sagemaker.DeleteEndpointConfigInput{
		EndpointConfigName: endpointDescription.EndpointConfigName,
	}); err != nil {
		return err
	}
	log.Infof("Deleted endpoint config=%s", *endpointDescription.EndpointConfigName)

	// Delete Model - this may take a few minutes
	if _, err := g.SagemakerClient.DeleteModel(context.TODO(), &sagemaker.DeleteModelInput{
		ModelName: aws.String(modelName),
	}); err != nil {
		return err
	}
	log.Infof("Deleted model=%s", modelName)

	// Avoid rate limiter.
	time.Sleep(sleepTime)

	return nil
}
