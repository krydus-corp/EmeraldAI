/*
 * File: dataset.go
 * Project: models
 * File Created: Saturday, 30th April 2022 11:37:00 am
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package models

import (
	"fmt"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	DefaultTrainSplit      = 0.75
	DefaultValidationSplit = 0.25
	DefaultTestSplit       = 0.00
)

// ErrInvalidDatasetSplit is an error return when an invalid train, validation, test split is specified
var ErrInvalidDatasetSplit = fmt.Errorf("train, validation, and test dataset split must sum to 1.0; train and validation splits cannot be 0")

// Dataset represents dataset domain model
//
// swagger:model Dataset
type Dataset struct {
	// ID of the Job
	//
	// swagger:strfmt bsonobjectid
	ID primitive.ObjectID `json:"id" bson:"_id"`
	// UserID associated with dataset
	//
	UserID string `json:"userid" bson:"userid"`
	// ProjectID associated with dataset
	//
	ProjectID string `json:"projectid" bson:"projectid"`
	// Dataset Split
	//
	Split struct {
		Train      float64 `json:"train" bson:"train"`
		Validation float64 `json:"validation" bson:"validation"`
		Test       float64 `json:"test" bson:"test"`
	} `json:"split" bson:"split"`
	// Dataset lock status - datasets created and attached to models are locked.
	// Datasets that are Locked cannot have tags or annotation deleted.
	//
	Locked bool `json:"locked" bson:"locked"`
	// Version of dataset
	//
	Version int `json:"version" bson:"version"`

	CreatedAt time.Time `json:"created_at" bson:"created_at" export:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

func NewDataset(userid, projectid string) *Dataset {

	id := primitive.NewObjectID()

	return &Dataset{
		ID:        id,
		UserID:    userid,
		ProjectID: projectid,
		Split: struct {
			Train      float64 "json:\"train\" bson:\"train\""
			Validation float64 "json:\"validation\" bson:\"validation\""
			Test       float64 "json:\"test\" bson:\"test\""
		}{DefaultTrainSplit, DefaultValidationSplit, DefaultTestSplit},
		Locked:    false,
		Version:   0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Valid validates dataset fields e.g. split
func (d *Dataset) Valid() error {
	if d.Split.Train == 0 || d.Split.Validation == 0 {
		return ErrInvalidDatasetSplit
	}
	if math.Round((d.Split.Train+d.Split.Validation+d.Split.Test)*1000000)/1000000 != float64(1.000000) {
		return ErrInvalidDatasetSplit
	}
	return nil
}

func (d *Dataset) IsZeroValue() bool {
	if d.Split.Train == 0 && d.Split.Validation == 0 && d.Split.Test == 0 {
		return true
	}
	return false
}
