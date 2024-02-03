/*
 * File: test_exporter.go
 * Project: exporter
 * File Created: Thursday, 5th January 2023 8:06:56 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 *
 * This is an integration test and requires a local instance of mongo with an existing  `admin` user.
 */
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"

	common "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	database "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
	exporter "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/exporter"
)

const (
	testEvent = `{
  "Records": [
    {
      "messageId" : "MessageID_1",
      "body" : "Message Body"
    }
  ]
}`

	testConfig = `---
database:
  url: mongodb://admin:emerald2022kIk59vEhxsNAY5Xk@localhost:27017
  timeout_seconds: 30

blob_store:
  bucket: emld-user-data-integration-test
`
)

var (
	testProject = fmt.Sprintf("integration-test-%s", common.ShortUUID(6))
	testExport  = fmt.Sprintf("integration-test-%s", common.ShortUUID(6))

	_ = (func() interface{} {
		_testing = true
		return nil
	}())
)

func TestSqsEventIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create config
	file, err := os.CreateTemp("", "emld-exporter-unit-cfg-*.yaml")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())

	if _, err := file.WriteString(testConfig); err != nil {
		panic(err)
	}

	os.Setenv("EXPORTER_CONFIG_PATH", file.Name())

	// Create test event
	var inputEvent events.SQSEvent
	if err := json.Unmarshal([]byte(testEvent), &inputEvent); err != nil {
		t.Fatalf("could not unmarshal event. details: %v", err)
	}

	// Initialize platform and DB
	plat := platform.NewPlatform()
	db, err := database.New("mongodb://admin:emerald2022kIk59vEhxsNAY5Xk@localhost:27017", 60)
	if err != nil {
		t.Fatal(err)
	}

	// Locate our admin user
	user, err := plat.UserDB.FindByUsername(db, "admin")
	if err != nil {
		t.Fatal(err)
	}

	// Create a new test project
	project, err := models.NewProject(user.ID.Hex(), testProject, "", "", "classification")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := plat.ProjectDB.Create(db, *project); err != nil {
		t.Fatal(err)
	}
	defer func() {
		plat.ProjectDB.Delete(db, user.ID.Hex(), project.ID.Hex())
	}()

	// Create a test export
	export := models.NewExport(user.ID.Hex(), testExport, "PROJECT")
	if _, err := plat.ExportDB.Create(db, export); err != nil {
		t.Fatal(err)
	}
	defer func() {
		plat.ExportDB.Delete(db, export.ID)
	}()

	// Update body with export message
	msg := exporter.Event{ExportID: export.ID.Hex(), UserID: user.ID.Hex()}
	msgBytes, _ := json.Marshal(msg)
	inputEvent.Records[0].Body = string(msgBytes)

	// Send for processing
	if err := HandleLambdaEvent(context.TODO(), inputEvent); err != nil {
		t.Fatalf("failed during event processing. details: %v", err)
	}
}
