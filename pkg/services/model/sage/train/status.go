/*
 * File: status.go
 * Project: train
 * File Created: Friday, 19th August 2022 6:17:52 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package train

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
	"github.com/aws/aws-sdk-go/aws"

	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
)

func (t *Trainer) PollForStatus() (*types.HyperParameterTrainingJobSummary, error) {
	var hyperParametersSummary *types.HyperParameterTrainingJobSummary

	for {
		output, err := t.Client.DescribeHyperParameterTuningJob(context.TODO(), &sagemaker.DescribeHyperParameterTuningJobInput{
			HyperParameterTuningJobName: aws.String(*t.Input.HyperParameterTuningJobName),
		})
		if err != nil {
			return nil, err
		}

		switch output.HyperParameterTuningJobStatus {
		case types.HyperParameterTuningJobStatusCompleted, types.HyperParameterTuningJobStatusStopped:
			hyperParametersSummary = output.BestTrainingJob
			return hyperParametersSummary, nil
		case types.HyperParameterTuningJobStatusInProgress:
			log.Debugf("training hyperparameter job; name=%s", *t.Input.HyperParameterTuningJobName)
			time.Sleep(5 * time.Second)
			continue
		case types.HyperParameterTuningJobStatusFailed:
			return nil, fmt.Errorf("training job error; error=%s", *output.FailureReason)
		case types.HyperParameterTuningJobStatusStopping:
			// Stop was initiated on the AWS console
			log.Debugf("training hyperparameter job stop was initiated; name=%s", *t.Input.HyperParameterTuningJobName)
			// Await full stop
			for {
				output, err = t.Client.DescribeHyperParameterTuningJob(context.TODO(), &sagemaker.DescribeHyperParameterTuningJobInput{
					HyperParameterTuningJobName: aws.String(*t.Input.HyperParameterTuningJobName),
				})
				if err != nil {
					return nil, err
				}
				switch output.HyperParameterTuningJobStatus {
				case types.HyperParameterTuningJobStatusStopped:
					return nil, fmt.Errorf("training job stopped; name=%s", *t.Input.HyperParameterTuningJobName)
				case types.HyperParameterTuningJobStatusFailed:
					return nil, fmt.Errorf("training job error; error=%s", *output.FailureReason)
				case types.HyperParameterTuningJobStatusStopping:
					log.Debugf("training job stopping; name=%s", *t.Input.HyperParameterTuningJobName)
					time.Sleep(1 * time.Second)
					continue
				default:
					return nil, fmt.Errorf("unexpected status for training job; status=%s", string(output.HyperParameterTuningJobStatus))
				}
			}
		default:
			return nil, fmt.Errorf("unexpected status for training job; status=%s", string(output.HyperParameterTuningJobStatus))
		}
	}
}
