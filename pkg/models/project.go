/*
 * File: project.go
 * Project: models
 * File Created: Monday, 15th March 2021 10:48:03 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package models

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ErrInvalidProjectAnnotation = echo.NewHTTPError(http.StatusBadRequest, "Invalid project annotation type. Acceptable options include: classification or bounding_box.")
var ErrAnnotationMismatch = echo.NewHTTPError(http.StatusBadRequest, "Annotation type does not match project annotation type")

type ProjectAnnotationType int

const (
	ProjectAnnotationTypeUnknown ProjectAnnotationType = iota
	ProjectAnnotationTypeClassification
	ProjectAnnotationTypeBoundingBox
)

func (a ProjectAnnotationType) String() string {
	return [...]string{"unknown", "classification", "bounding_box"}[a]
}

func ProjectAnnotationTypeFromString(str string) (ProjectAnnotationType, error) {
	switch strings.ToLower(str) {

	case "classification":
		return ProjectAnnotationTypeClassification, nil
	case "bounding_box":
		return ProjectAnnotationTypeBoundingBox, nil
	default:
		return ProjectAnnotationTypeUnknown, ErrInvalidProjectAnnotation
	}
}

// Project represents project domain model
//
// swagger:model Project
type Project struct {
	// ID of the Project
	//
	// swagger:strfmt bsonobjectid
	ID primitive.ObjectID `json:"id" bson:"_id" export:"id"`
	// Dataset ID
	//
	DatasetID string `json:"datasetid" bson:"datasetid"`
	// UserID associated with Project
	//
	UserID string `json:"userid" bson:"userid"`
	// Name of Project
	//
	Name string `json:"name" bson:"name" export:"name"`
	// Description of Project
	//
	Description *string `json:"description" bson:"description"`
	// Project item count
	//
	Count int64 `json:"count" bson:"count" export:"count"`
	// Annotation type
	//
	AnnotationType string `json:"annotation_type" bson:"annotation_type" export:"annotation_type"`
	// License type
	//
	LicenseType *string `json:"license" bson:"license" export:"license"`
	// Project profile picture (100x100 px)
	//
	Profile100 string `json:"profile_100" bson:"profile_100"`
	// Project profile picture (200x200 px)
	//
	Profile200 string `json:"profile_200" bson:"profile_200"`
	// Project profile picture (640x640 px)
	//
	Profile640 string `json:"profile_640" bson:"profile_640"`

	CreatedAt time.Time `json:"created_at" bson:"created_at" export:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at" export:"updated_at"`
}

func NewProject(userid, projectName, projectDescription, license, annotation string) (*Project, error) {
	annotationType, err := ProjectAnnotationTypeFromString(annotation)
	if err != nil {
		return nil, err
	}

	return &Project{
		ID:          primitive.NewObjectID(),
		UserID:      userid,
		Name:        projectName,
		Description: &projectDescription,
		Count:       0,

		AnnotationType: annotationType.String(),
		LicenseType:    &license,

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}
