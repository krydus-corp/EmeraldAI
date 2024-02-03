/*
 * File: export.go
 * Project: models
 * File Created: Friday, 12th November 2021 10:39:33 am
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package models

import (
	"fmt"
	"path"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ExportState int

const (
	ExportStateUnknown ExportState = iota
	ExportStateInitialized
	ExportStateRunning
	ExportStateComplete
	ExportStateErr
)

func (s ExportState) String() string {
	return [...]string{"UNKNOWN", "INITIALIZED", "RUNNING", "COMPLETE", "ERR"}[s]
}

type ExportType string

const (
	ExportTypeUnknown ExportType = "UNKNOWN"
	ExportTypeProject ExportType = "PROJECT"
	ExportTypeDataset ExportType = "DATASET"
	ExportTypeModel   ExportType = "MODEL"
)

// Export represents export domain model
//
// swagger:model Export
type Export struct {
	// ID of the Export
	//
	// swagger:strfmt bsonobjectid
	ID primitive.ObjectID `json:"id" bson:"_id"`
	// Name of Export
	//
	Name string `json:"name" bson:"name" export:"name"`
	// Type of export -> maps the type to the id
	//
	Type ExportType `json:"type" bson:"type" export:"type"`
	// UserID associated with Export
	//
	UserID string `json:"userid" bson:"userid" export:"userid"`
	// Export blobstore path
	//
	Path string `json:"path" bson:"path"`
	// State of the Export
	//
	State string `json:"state" bson:"state"`
	// Last error associated with this Export
	//
	LastError string `json:"error" bson:"error"`
	// Content keys
	//
	ContentKeys []string `json:"content_keys" bson:"content_keys" export:"content_keys"`
	// Metadata - populated based on export type
	//
	Project *Project `json:"project,omitempty" bson:"project,omitempty" export:"project"`
	Dataset *Dataset `json:"dataset,omitempty" bson:"dataset,omitempty" export:"dataset"`
	Model   *Model   `json:"model,omitempty" bson:"model,omitempty" export:"model"`

	Metadata map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty" export:"metadata"`

	ErrorAt   time.Time `json:"error_at" bson:"error_at"`
	CreatedAt time.Time `json:"created_at" bson:"created_at" export:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at" export:"updated_at"`
}

func NewExport(userid, name string, exportType ExportType) Export {
	id := primitive.NewObjectID()

	export := Export{
		ID:          id,
		Name:        name,
		UserID:      userid,
		State:       ExportStateInitialized.String(),
		Type:        exportType,
		Path:        path.Join(userid, "exports", id.Hex()),
		ContentKeys: []string{},

		ErrorAt:   time.Time{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return export
}

func (e *Export) UpdateMetadata(project *Project, model *Model, dataset *Dataset) (err error) {
	if project != nil {
		e.Project = project
	} else if model != nil {
		e.Model = model
	} else if dataset != nil {
		e.Dataset = dataset
	} else {
		err = fmt.Errorf("one of project, model, or dataset must not be nil")
	}

	return
}
