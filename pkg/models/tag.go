/*
 * File: tag.go
 * Project: models
 * File Created: Tuesday, 11th May 2021 2:20:16 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package models

import (
	"strings"
	"time"

	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TagType int

const (
	TagTypeHuman TagType = iota
	TagTypeMachine
)

func (t TagType) String() string {
	return [...]string{"HUMAN", "MACHINE"}[t]
}

// Tag represents tag domain model
//
// swagger:model Tag
type Tag struct {
	// ID of the Tag
	//
	// swagger:strfmt bsonobjectid
	ID primitive.ObjectID `json:"id" bson:"_id"`
	// UserID associated with Tag
	//
	UserID string `json:"userid" bson:"userid"`
	// ProjectID associated with Tag
	//
	ProjectID string `json:"projectid" bson:"projectid"`
	// DatasetID associated with Tag
	//
	DatasetID string `json:"datasetid" bson:"datasetid"`
	// Name of Tag
	//
	Name string `json:"name" bson:"name"`
	// Tag properties
	//
	Property []string `json:"property" bson:"property"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

func NewTag(userid, projectid, datasetid, name string, properties []string) Tag {
	return Tag{
		ID:        primitive.NewObjectID(),
		UserID:    userid,
		ProjectID: projectid,
		DatasetID: datasetid,

		Name:     strings.ToLower(name),
		Property: common.RemoveDuplicateStr(common.StringSliceToLower(properties)),

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
