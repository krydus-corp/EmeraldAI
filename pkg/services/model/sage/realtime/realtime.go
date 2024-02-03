/*
 * File: spec.go
 * Project: realtime
 * File Created: Tuesday, 13th July 2021 3:29:30 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package realtime

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sagemakerruntime"
	"github.com/aws/aws-sdk-go/aws"

	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
)

func New(fileBytes []byte, modelEndpoint string, modelType string, client *sagemakerruntime.Client) (ModelReturn, error) {
	endpointOutput, err := invoke(fileBytes, modelEndpoint, client)
	if err != nil {
		return nil, err
	}
	// Predictions
	if modelType == models.ProjectAnnotationTypeClassification.String() {
		output := ClassificationResult{}
		if err := json.Unmarshal(endpointOutput.Body, &output.Return); err != nil {
			return nil, err
		}
		return &output, nil
	}

	if modelType == models.ProjectAnnotationTypeBoundingBox.String() {
		output := ObjectDetectionResult{}
		if err := json.Unmarshal(endpointOutput.Body, &output.Return); err != nil {
			return nil, err
		}
		return &output, nil
	}
	return nil, fmt.Errorf("unrecognized model type")
}

func invoke(fileBytes []byte, modelEndpoint string, client *sagemakerruntime.Client) (*sagemakerruntime.InvokeEndpointOutput, error) {
	endpointOutput, err := client.InvokeEndpoint(context.TODO(), &sagemakerruntime.InvokeEndpointInput{
		EndpointName: aws.String(modelEndpoint),
		ContentType:  aws.String("application/x-image"),
		Body:         fileBytes,
	})
	if err != nil {
		return nil, err
	}
	return endpointOutput, nil
}
