/*
 * File: status.go
 * Project: endpoint
 * File Created: Friday, 19th August 2022 7:38:00 pm
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
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
)

func (e *Endpoint) PollForStatus() error {
	// Poll for status
	for {
		status, err := e.Client.DescribeEndpoint(context.TODO(), &sagemaker.DescribeEndpointInput{
			EndpointName: aws.String(e.endpointName),
		})
		if err != nil {
			return err
		}

		switch status.EndpointStatus {
		case types.EndpointStatusInService:
			return nil
		case types.EndpointStatusCreating:
			log.Debugf("creating endpoint; name=%s", e.endpointName)
			time.Sleep(1 * time.Second)
			continue
		case types.EndpointStatusFailed:
			return fmt.Errorf("endpoint job error; error=%s", *status.FailureReason)
		default:
			return fmt.Errorf("unexpected status for endpoint job; status=%s", status.EndpointStatus)
		}
	}
}
