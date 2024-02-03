/*
 * File: annotation.go
 * Project: models
 * File Created: Monday, 2nd May 2022 7:02:17 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package models

import (
	"fmt"
	"strings"
	"time"

	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Split is an enum for split type
type Split int

const (
	SplitUndefined Split = iota
	SplitTrain
	SplitValidation
	SplitTest
)

func (s Split) String() string {
	return [...]string{"UNDEFINED", "TRAIN", "VALIDATION", "TEST"}[s]
}

type AnnotationDataBoundingBox struct {
	TagID string `json:"name,omitempty" bson:"tagid,omitempty"`
	Xmin  int    `json:"xmin,omitempty" bson:"xmin,omitempty"`
	Ymin  int    `json:"ymin,omitempty" bson:"ymin,omitempty"`
	Xmax  int    `json:"xmax,omitempty" bson:"xmax,omitempty"`
	Ymax  int    `json:"ymax,omitempty" bson:"ymax,omitempty"`
}

// ToTopLeftWidthHeightFormat converts to left/top, height/width format
// left and top identify the location of the pixel in the top-left corner of the bounding box relative to the top-left corner of the image.
// The dimensions of the bounding box are identified with height and width.
func (a AnnotationDataBoundingBox) ToTopLeftWidthHeightFormat() (left int, top int, width int, height int) {
	left = a.Xmin
	top = a.Ymin
	width = a.Xmax - a.Xmin
	height = a.Ymax - a.Ymin
	return
}

type AnnotationMetadata struct {
	BoundingBoxes []AnnotationDataBoundingBox `json:"bounding_boxes,omitempty" bson:"bounding_boxes,omitempty"`
}

type ContentMetadata struct {
	Size   int `json:"size" bson:"size"`
	Height int `json:"height" bson:"height"`
	Width  int `json:"width" bson:"width"`
}

// Annotation represents annotation domain model
//
// swagger:model Annotation
type Annotation struct {
	// ID of the Annotation
	//
	// swagger:strfmt bsonobjectid
	ID primitive.ObjectID `json:"id" bson:"_id"`
	// UserID associated with Annotation
	//
	UserID string `json:"userid" bson:"userid"`
	// ProjectID associated with Annotation
	//
	ProjectID string `json:"projectid" bson:"projectid"`
	// Dataset associated with this annotation
	//
	DatasetID string `json:"datasetid" bson:"datasetid"`
	// Tag IDs
	//
	TagIDs []string `json:"tagids" bson:"tagids"`
	// Content ID
	//
	ContentID string `json:"contentid" bson:"contentid"`
	// Train, validation, or test label
	//
	Split string `json:"split" bson:"split"`
	// Content b64 string
	//
	Base64Image string `json:"b64_image" bson:"b64_image"`
	// Annotation metadata
	//
	Metadata AnnotationMetadata `json:"metadata,omitempty" bson:"metadata,omitempty"`
	// Content metadata
	//
	ContentMetadata ContentMetadata `json:"-" bson:"content_metadata"`
	// Flag indicating null annotation
	//
	IsNullAnnotation bool `json:"-" bson:"null_annotation"`
	// Content associated with annotation
	// This is never populated. It is used in Mongo aggregation pipelines when joining the annotation
	// collection with the content collection so that we can locate all content with an annotation.
	Content []Content `json:"-" bson:"content"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

func NewAnnotation(userid, projectid, datasetid, contentid string, tagids []string, base64Image string, metadata AnnotationMetadata, contentMetadata ContentMetadata) *Annotation {
	isNullAnnotation := false
	if len(tagids) == 0 {
		isNullAnnotation = true
	}

	return &Annotation{
		ID:               primitive.NewObjectID(),
		UserID:           userid,
		ProjectID:        projectid,
		DatasetID:        datasetid,
		TagIDs:           tagids,
		ContentID:        contentid,
		Split:            SplitUndefined.String(),
		Base64Image:      base64Image,
		Metadata:         metadata,
		ContentMetadata:  contentMetadata,
		IsNullAnnotation: isNullAnnotation,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

func (a *Annotation) Valid(annotationType string) error {
	projectAnnotationType, err := ProjectAnnotationTypeFromString(annotationType)
	if err != nil {
		return err
	}

	switch projectAnnotationType {
	case ProjectAnnotationTypeBoundingBox:

		if len(a.Metadata.BoundingBoxes) > 0 {
			boundingBoxTagIdSet := make(map[string]struct{})

			for _, box := range a.Metadata.BoundingBoxes {
				if box.TagID == "" {
					return fmt.Errorf("annotation bounding box has nil tagid")
				}
				if box.Xmax == 0 && box.Xmin == 0 && box.Ymax == 0 && box.Ymin == 0 {
					return fmt.Errorf("annotation bounding box contains all zero values")
				}
				boundingBoxTagIdSet[box.TagID] = struct{}{}
			}

			if len(boundingBoxTagIdSet) != len(a.TagIDs) {
				return fmt.Errorf(
					"annotation tagids do not match tagids found in bounding-box metadata; annotation-tagids=%s, bounding-box-tagids=%s",
					strings.Join(common.MapStringStructToSlice(boundingBoxTagIdSet), ","),
					strings.Join(a.TagIDs, ","),
				)
			}
		}
	case ProjectAnnotationTypeClassification:
		// Noop
	case ProjectAnnotationTypeUnknown:
		return ErrInvalidProjectAnnotation
	}

	return nil
}
