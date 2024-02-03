/*
 * File: export.go
 * Project: export
 * File Created: Tuesday, 14th September 2021 7:19:43 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package export

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	common "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	awssqs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/aws/sqs"
	worker "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/worker"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	exporterErr "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/exporter/error"
	exporter "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/exporter/worker"
)

// Create creates a new export entry
func (e Export) Create(c echo.Context, userid string, req createExportReq) (models.Export, error) {

	exportType := models.ExportType(strings.ToUpper(req.ExportType))
	export := models.NewExport(userid, req.Name, exportType)

	// Initialize the specified type of export
	switch exportType {
	case models.ExportTypeProject:
		project, err := e.platform.ProjectDB.View(e.db, userid, req.ID)
		if err != nil {
			return models.Export{}, echo.NewHTTPError(404, fmt.Sprintf("unable to locate projectid=%s", req.ID))
		}
		export.UpdateMetadata(&project, nil, nil)
	case models.ExportTypeDataset:
		dataset, err := e.platform.DatasetDB.View(e.db, userid, req.ID)
		if err != nil {
			return models.Export{}, echo.NewHTTPError(404, fmt.Sprintf("unable to locate datasetid=%s", req.ID))
		}
		export.UpdateMetadata(nil, nil, dataset)
	default:
		return models.Export{}, echo.NewHTTPError(http.StatusNotImplemented, "export type not supported")
	}

	// Database operation
	if _, err := e.platform.ExportDB.Create(e.db, export); err != nil {
		return models.Export{}, errors.Wrap(err, "unable to create export")
	}

	message := fmt.Sprintf(`{"export_id":"%s","user_id":"%s"}`, export.ID.Hex(), userid)

	// Invoke lambda functions directly (local deployment)
	// FIXME: Importing and using this here causing the `initialize()` func in the exporter package to trigger,
	// resulting in unnecessary initializations. Maybe we can remove at compile time???
	if strings.ToLower(os.Getenv("AWS_LAMBDA_EXECUTION_ENV_LOCAL")) == "true" {
		os.Setenv("EXPORTER_CONFIG_PATH", fmt.Sprintf("s3://emld-configuration-store/config.exporter.%s.yml", os.Getenv("CLUSTER_ENV")))

		event := events.SQSEvent{Records: []events.SQSMessage{
			{MessageId: common.ShortUUID(6), Body: message},
		}}

		if err := exporter.HandleLambdaEvent(context.TODO(), event); err != nil {
			// Simulate DLQ by sending to lambda error handler in the exporter service
			errMessage := fmt.Sprintf(`{"export_id":"%s","user_id":"%s", "error":"%s"}`, export.ID.Hex(), userid, err.Error())
			event := events.SQSEvent{Records: []events.SQSMessage{
				{MessageId: common.ShortUUID(6), Body: errMessage},
			}}

			if err := exporterErr.HandleLambdaEvent(context.TODO(), event); err != nil {
				return models.Export{}, errors.Wrap(err, "export error handling error")
			}
		}
		return e.platform.ExportDB.View(e.db, userid, export.ID.Hex())
	}

	// Send job over to lambda event queue (prod deployment)
	if _, err := worker.Retry(3, true, true, 2, func() (struct{}, error) {
		return struct{}{}, e.publisher.Publish(context.TODO(), awssqs.JSONString(message))
	}); err != nil {
		return models.Export{}, errors.Wrap(err, "unable to queue export")
	}

	return export, nil
}

func (e Export) View(c echo.Context, userid, exportid string) (models.Export, error) {
	export, err := e.platform.ExportDB.View(e.db, userid, exportid)
	if err != nil {
		return models.Export{}, err
	}

	return export, nil
}

func (e Export) List(c echo.Context, userid string, p models.Pagination) ([]models.Export, int64, error) {
	return e.platform.ExportDB.List(e.db, userid, p)
}

func (e Export) Query(ctx echo.Context, userid string, q models.Query) ([]models.Export, int64, error) {
	return e.platform.ExportDB.Query(e.db, userid, q)
}

// Delete deletes the export associated with a project and user
func (e Export) Delete(c echo.Context, userid, exportid string) error {
	// delete export metadata
	export, err := e.platform.ExportDB.View(e.db, userid, exportid)
	if err != nil {
		return err
	}

	if err := e.platform.ExportDB.Delete(e.db, export.ID); err != nil {
		return nil
	}

	// delete export blobs
	e.blob.DeleteChan <- export.Path

	return nil
}

// GetContent retrieves the content for an export
func (e Export) GetContent(c echo.Context, userid, exportid, contentURI string, expireMinutes int) (string, error) {
	export, err := e.platform.ExportDB.View(e.db, userid, exportid)
	if err != nil {
		return "", err
	}

	// Check if content URI belong to user's export
	exportContentURI := false
	for _, uri := range export.ContentKeys {
		if contentURI == uri {
			exportContentURI = true
			break
		}
	}

	if !exportContentURI {
		return "", echo.NewHTTPError(404, "content URI does not exists")
	}

	req, _ := e.blob.S3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(e.blob.Bucket),
		Key:    aws.String(contentURI),
	})

	url, err := req.Presign(time.Duration(expireMinutes) * time.Minute)
	if err != nil {
		return "", err
	}

	return url, nil
}

// GetMetadata retrieves the metadata for an export
func (e Export) GetMetadata(c echo.Context, userid, exportid string) ([]byte, error) {
	export, err := e.platform.ExportDB.View(e.db, userid, exportid)
	if err != nil {
		return nil, err
	}

	exportMap, err := common.SelectStructFields(export, "export")
	if err != nil {
		return nil, err
	}

	b, err := json.MarshalIndent(exportMap, "", " ")
	if err != nil {
		return nil, err
	}

	return b, nil
}
