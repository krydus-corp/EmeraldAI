/*
 * File: upload.go
 * Project: models
 * File Created: Monday, 16th May 2022 2:30:20 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UploadState int

const (
	UploadStateUnknown UploadState = iota
	UploadStateInitialized
	UploadStateRunning
	UploadStateComplete
	UploadStateCompleteWithErr
	UploadStateErr
)

func (s UploadState) String() string {
	return [...]string{"UNKNOWN", "INITIALIZED", "RUNNING", "COMPLETE", "COMPLETE_WITH_ERR", "ERR"}[s]
}

type UploadErr int

const (
	UploadErrUnknown UploadErr = iota
	UploadErrS3
	UploadErrInvalidRequest
	UploadErrCreatingAssets
	UploadErrImageStats
	UploadErrId
	UploadErrLabel
	UploadErrMaxSize
	UploadErrAlreadyExists
	UploadErrUpdate
)

func (s UploadErr) String() string {
	return [...]string{"UNKNOWN_ERROR", "S3_ERROR", "INTERNAL_REQUEST_ERROR", "ASSET_CREATION_ERROR", "INVALID_IMAGE_ERROR", "INTERNAL_ID_ERROR", "LABEL_ASSIGNMENT_ERROR", "EXCEEDS_MAX_IMAGE_SIZE_ERROR", "ALREADY_EXISTS_ERROR", "INTERNAL_UPDATE_ERROR"}[s]
}

// Upload represents upload job
//
// swagger:model Upload
type Upload struct {
	// ID of the Upload
	//
	// swagger:strfmt bsonobjectid
	ID primitive.ObjectID `json:"id" bson:"_id"`
	// UserID associated with Upload
	//
	UserID string `json:"userid" bson:"userid"`
	// ProjectID associated with Upload
	//
	ProjectID string `json:"projectid" bson:"projectid"`
	// State of the Upload
	//
	State string `json:"state" bson:"state"`
	// Bucket
	//
	Bucket string `json:"bucket" bson:"bucket"`
	// Key of upload
	//
	Key string `json:"key" bson:"key"`
	// Labels file associated with Upload
	//
	LabelsFile string `json:"labels_file" bson:"labels_file"`
	// Files that failed to upload
	//
	Failed map[string]interface{} `json:"failed" bson:"failed"`
	// Catastrophic error that occurred before file processing
	//
	Error string `json:"error" bson:"error"`
	// Total bytes
	//
	TotalBytes int64 `json:"total_bytes" bson:"total_bytes"`
	// Total bytes uploaded
	//
	TotalBytesUploaded int64 `json:"total_bytes_uploaded" bson:"total_bytes_uploaded"`
	// Total file uploaded
	//
	TotalFileUploaded int64 `json:"total_file_uploaded" bson:"total_file_uploaded"`
	// Total images processed
	//
	TotalImages int `json:"total_images" bson:"total_images"`
	// Total images that failed
	//
	TotalImagesFailed int `json:"total_images_failed" bson:"total_images_failed"`
	// Total images that succeeded
	//
	TotalImagesSucceeded int `json:"total_images_succeeded" bson:"total_images_succeeded"`
	// Processed images
	//
	TotalImagesProcessed int `json:"total_images_processed" bson:"total_images_processed"`
	// Total duplicate images
	//
	TotalImagesDuplicate int `json:"total_images_duplicate" bson:"total_images_duplicate"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`

	// Upload metadata
	//
	Metadata map[string]interface{} `json:"metadata" bson:"metadata"`
}

func NewUpload(userid, projectid, bucket, labelsFile string) Upload {

	return Upload{
		ID:         primitive.NewObjectID(),
		UserID:     userid,
		ProjectID:  projectid,
		State:      UploadStateInitialized.String(),
		Key:        fmt.Sprintf("%s/content/staging/%s/%s", userid, projectid, uuid.New()),
		LabelsFile: labelsFile,
		Bucket:     bucket,
		Failed:     make(map[string]interface{}),

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),

		Metadata: make(map[string]interface{}),
	}
}
