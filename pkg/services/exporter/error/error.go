/*
 * File: error.go
 * Project: error
 * File Created: Friday, 6th January 2023 10:31:06 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package error

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/pkg/errors"

	config "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/config"
	database "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
	exporter "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/exporter"
)

var (
	cfg  *exporter.Configuration
	plat *platform.Platform
	db   *database.DB

	err      error
	_testing = false
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
}

func HandleLambdaEvent(ctx context.Context, sqsEvent events.SQSEvent) error {
	if _testing {
		initialize()
	}

	for _, record := range sqsEvent.Records {

		// TODO: get error message from record - not sure how this is included in the event from lambda
		b, _ := json.MarshalIndent(record, "", " ")
		fmt.Println(string(b))

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

		// Update error state
		export.LastError = event.Error
		export.State = models.ExportStateErr.String()
		export.UpdatedAt, export.ErrorAt = time.Now(), time.Now()
		err = plat.ExportDB.Update(db, export)
		if err != nil {
			return errors.Wrapf(err, "error updating export metadata; export=%s", export.ID.Hex())
		}
	}

	return nil
}
