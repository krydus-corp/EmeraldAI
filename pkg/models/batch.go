/*
 * File: batch.go
 * Project: models
 * File Created: Sunday, 11th September 2022 1:43:10 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package models

import "time"

type BatchStatus int

const (
	BatchStatusUnknown BatchStatus = iota
	BatchStatusInitialized
	BatchStatusRunning
	BatchStatusComplete
	BatchStatusCompleteWithErr
	BatchStatusErr
)

func (d BatchStatus) String() string {
	return [...]string{"UNKNOWN", "INITIALIZED", "RUNNING", "COMPLETE", "COMPLETE_WITH_ERR", "ERR"}[d]
}

// Batch represents a batch apply job, which consist of predictions for all unannotated data in a project.
//
// swagger:model Batch
type Batch struct {
	// Status of the job
	//
	Status string `json:"status" bson:"status"`
	// Job started at time
	//
	StartedAt time.Time `json:"started_at" bson:"started_at"`
	// Job ended at time
	//
	EndedAt time.Time `json:"ended_at" bson:"ended_at"`
	// Total content to process
	//
	TotalContent int `json:"total_content" bson:"total_content"`
	// Completed content
	//
	CompletedContent int `json:"completed_content" bson:"completed_content"`
	// Min confidence threshold
	//
	Threshold float64 `json:"threshold" bson:"threshold"`
	// Last error (if any) associated with the batch job
	//
	LastError *string `json:"error" bson:"error"`
}

func NewBatch() Batch {
	return Batch{
		Status:           BatchStatusInitialized.String(),
		StartedAt:        time.Now(),
		EndedAt:          time.Time{},
		TotalContent:     0,
		CompletedContent: 0,
		Threshold:        0,
		LastError:        nil,
	}
}
