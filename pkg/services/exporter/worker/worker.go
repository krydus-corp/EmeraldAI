/*
 * File: exporter.go
 * Project: exporter
 * File Created: Thursday, 5th January 2023 5:41:36 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package worker

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"

	zipwriter "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/archive"
	s3 "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/aws/s3"
	config "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/config"
	database "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
	exporter "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/exporter"
)

var (
	cfg          *exporter.Configuration
	plat         *platform.Platform
	db           *database.DB
	zipw         *zipwriter.ZipWriter
	s3downloader *s3.S3Downloader
	s3uploader   *s3.S3Uploader

	err      error
	_testing = false
)

const (
	Const_S3Concurrency = 10
	Const_S3Partsize    = 1024 * 1024 * 5 // 5 MB
)

func init() {
	local := strings.ToLower(os.Getenv("AWS_LAMBDA_EXECUTION_ENV_LOCAL"))
	if _testing || local == "false" {
		return
	}
	initialize()
}

func initialize() {
	// Initialize config
	log.Println("Initializing config...")
	path, exists := os.LookupEnv("EXPORTER_CONFIG_PATH")
	if !exists {
		panic("required environmental variable `EXPORTER_CONFIG_PATH` unset")
	}

	cfg, err = config.NewConfiguration[exporter.Configuration](path, strings.HasPrefix(path, "s3://"))
	if err != nil {
		panic("Configuration initialization error; " + err.Error())
	}

	// Initialize platform
	plat = platform.NewPlatform()

	// Initialize DB
	log.Println("Initializing DB...")
	db, err = database.New(cfg.DB.URL, cfg.DB.Timeout)
	if err != nil {
		panic("DB initialization error; " + err.Error())
	}

	// Initialize an S3 downloader
	log.Println("Initializing downloader...")
	s3downloader, err = s3.NewS3Downloader(Const_S3Partsize, Const_S3Concurrency, cfg.BlobStore.Bucket)
	if err != nil {
		panic("S3 downloader initialization error; " + err.Error())
	}

	// Initialize an S3 uploader
	log.Println("Initializing uploader...")
	s3uploader, err = s3.NewS3Uploader(Const_S3Partsize, Const_S3Concurrency, cfg.BlobStore.Bucket)
	if err != nil {
		panic("S3 uploader initialization error; " + err.Error())
	}

	// Initialize zipwriter
	log.Println("Initializing zipwriter...")
	zipw = zipwriter.New(
		s3downloader.Downloader,
		s3uploader.Uploader,
	)

	log.Println("Initializing complete.")
}

func HandleLambdaEvent(ctx context.Context, sqsEvent events.SQSEvent) error {
	if _testing {
		initialize()
	}

	for _, record := range sqsEvent.Records {

		log.Printf("Received message with body: %s", record.Body)

		// Unmarshal event
		var event exporter.Event
		if err := json.Unmarshal([]byte(record.Body), &event); err != nil {
			return err
		}

		// Retrieve export associated with this event
		export, err := plat.ExportDB.View(db, event.UserID, event.ExportID)
		if err != nil {
			return err
		}

		// Set export state to running
		export.State = models.ExportStateRunning.String()
		if err := plat.ExportDB.Update(db, export); err != nil {
			return err
		}

		// Run export
		exportPackage := NewExport(db, plat, zipw, cfg.BlobStore.Bucket)
		if err = exportPackage.Export(&export); err != nil {
			return err
		}

		// Update state
		export.State = models.ExportStateComplete.String()
		if err := plat.ExportDB.Update(db, export); err != nil {
			return err
		}
	}

	return nil
}
