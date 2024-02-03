/*
 * File: prediction.go
 * Project: models
 * File Created: Monday, 5th September 2022 7:08:09 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Prediction represents prediction domain model
//
// swagger:model Prediction
type Prediction struct {
	// ID of the prediction
	//
	// swagger:strfmt bsonobjectid
	ID primitive.ObjectID `json:"id" bson:"_id"`
	// UserID associated with prediction
	//
	UserID string `json:"userid" bson:"userid"`
	// ModelID associated with prediction
	//
	ModelID string `json:"modelid" bson:"modelid"`
	// Content ID
	//
	ContentID string `json:"contentid" bson:"contentid"`
	// Prediction type
	//
	Type string `json:"type" bson:"type"`
	// Content b64 string
	//
	Base64Image string `json:"b64_image" bson:"b64_image"`
	// Predictions
	//
	Predictions []PredictionMetadata `json:"predictions" bson:"predictions"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

// Formatting, distinction between classification/object_detection
//
//	should be on the bounding-box members presence
type PredictionMetadata struct {
	ClassIndex  int                    `json:"class_index" bson:"class_index"`
	ClassName   string                 `json:"class_name" bson:"class_name"`
	Confidence  float64                `json:"confidence" bson:"confidence"`
	BoundingBox map[string]interface{} `json:"bounding_boxes" bson:"bounding_boxes"`

	TagID string `json:"tagid" bson:"-,omitempty"`
}

func NewPrediction(userid, modelid, contentid string, predictionType ProjectAnnotationType, base64Img string, predictions []PredictionMetadata) Prediction {
	return Prediction{
		ID:          primitive.NewObjectID(),
		UserID:      userid,
		ModelID:     modelid,
		ContentID:   contentid,
		Type:        predictionType.String(),
		Base64Image: base64Img,
		Predictions: predictions,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
