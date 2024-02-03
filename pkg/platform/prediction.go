/*
 * File: prediction.go
 * Project: platform
 * File Created: Sunday, 11th September 2022 8:33:49 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package platform

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	common "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
)

// Custom errors
var (
	ErrPredictionAlreadyExists = echo.NewHTTPError(http.StatusConflict, "Prediction already exists.")
	ErrPredictionDoesNotExist  = echo.NewHTTPError(http.StatusNotFound, "Prediction does not exists.")
)

// Prediction represents the client for prediction table
type Prediction struct{}

func NewPrediction() *Prediction {
	return &Prediction{}
}

// PredictionDB represents Prediction repository interface
type PredictionDB interface {
	Index(*db.DB) error
	Create(*db.DB, []models.Prediction) (int, error)
	Query(*db.DB, string, models.Query) ([]models.Prediction, int64, error)
	DeleteMany(*db.DB, string, string) error
	Sample(*db.DB, string, string, []string, float64, int) ([]models.Prediction, error)
	PredictionsPerClass(*db.DB, models.Model, float64, ...string) ([]map[string]interface{}, error)
	View(*db.DB, string, string) (*models.Prediction, error)
}

func (p Prediction) Index(db *db.DB) error {
	collection := db.Client.Database(DATABASE).Collection(PREDICTION_COLLECTION)

	models := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "_id", Value: 1}, {Key: "userid", Value: 1}},
			Options: &options.IndexOptions{Unique: common.Ptr(true), Background: common.Ptr(true)},
		},
		{
			Keys:    bson.D{{Key: "userid", Value: 1}, {Key: "modelid", Value: 1}, {Key: "contentid", Value: 1}},
			Options: &options.IndexOptions{Unique: common.Ptr(true), Background: common.Ptr(true)},
		},
	}

	if _, err := collection.Indexes().CreateMany(context.TODO(), models); err != nil {
		return err
	}
	return nil
}

func (p Prediction) PredictionsPerClass(db *db.DB, model models.Model, threshold float64, tagIDs ...string) ([]map[string]interface{}, error) {
	collection := db.Client.Database(DATABASE).Collection(PREDICTION_COLLECTION)

	var lookup bson.M

	if len(tagIDs) > 0 {
		objectTags := []primitive.ObjectID{}
		for _, t := range tagIDs {
			objectTag, err := primitive.ObjectIDFromHex(t)
			if err != nil {
				return nil, err
			}
			objectTags = append(objectTags, objectTag)
		}

		lookup = bson.M{
			"from": TAG_COLLECTION,
			"pipeline": []bson.M{
				{"$match": bson.M{
					"_id":       bson.M{"$in": objectTags},
					"userid":    model.UserID,
					"datasetid": model.DatasetID,
				}},
				{"$project": bson.M{
					"_id":  1,
					"name": "$name",
				}},
			},
			"as": "tags",
		}
	} else {
		lookup = bson.M{
			"from": TAG_COLLECTION,
			"pipeline": []bson.M{
				{"$match": bson.M{
					"userid":    model.UserID,
					"datasetid": model.DatasetID,
				}},
				{"$project": bson.M{
					"_id":  1,
					"name": "$name",
				}},
			},
			"as": "tags",
		}
	}

	pipeline := []bson.M{
		{"$match": bson.M{
			"userid":  model.UserID,
			"modelid": model.ID.Hex(),
		}},
		{"$unwind": "$predictions"},
		{"$lookup": lookup},
		{"$addFields": bson.M{
			"tagFilter": bson.M{
				"$arrayElemAt": bson.A{
					bson.M{"$filter": bson.M{
						"input": "$tags",
						"as":    "tags",
						"cond": bson.M{
							"$eq": bson.A{"$$tags.name", "$predictions.class_name"},
						},
					}}, 0,
				},
			},
		},
		},
		{"$match": bson.M{
			"predictions.confidence": bson.M{
				"$gt": threshold,
			},
			"tagFilter._id": bson.M{
				"$exists": true,
				"$ne":     nil,
			},
		}},
		{"$group": bson.M{"_id": "$predictions.class_name", "count": bson.M{"$sum": 1}, "tagid": bson.M{"$first": "$tagFilter._id"}}},
		{"$project": bson.M{
			"_id":       0,
			"classname": "$_id",
			"count":     "$count",
			"tagid":     "$tagid",
		}},
	}

	var results []map[string]interface{}

	cursor, err := collection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, err
	}

	if err := cursor.All(context.TODO(), &results); err != nil {
		return nil, err
	}

	return results, nil
}

func (p Prediction) Sample(db *db.DB, userid, modelid string, tagNames []string, threshold float64, sampleCount int) ([]models.Prediction, error) {
	collection := db.Client.Database(DATABASE).Collection(PREDICTION_COLLECTION)

	var predictions = []models.Prediction{}
	var tags bson.A

	for _, name := range tagNames {
		tags = append(tags, name)
	}

	// TODO: to make this more efficient, we should apply the sample after match and then loop until we get the desired count
	pipeline := bson.A{
		bson.D{
			{Key: "$match",
				Value: bson.D{
					{Key: "userid", Value: userid},
					{Key: "modelid", Value: modelid},
				},
			},
		},
		bson.D{{Key: "$unwind", Value: bson.D{{Key: "path", Value: "$predictions"}}}},
		bson.D{
			{Key: "$match",
				Value: bson.D{
					{Key: "predictions.class_name",
						Value: bson.D{
							{Key: "$in", Value: tags},
						},
					},
					{Key: "predictions.confidence", Value: bson.D{{Key: "$gt", Value: threshold}}},
				},
			},
		},
		bson.D{
			{Key: "$group",
				Value: bson.D{
					{Key: "_id", Value: "$_id"},
					{Key: "userid", Value: bson.D{{Key: "$last", Value: "$userid"}}},
					{Key: "modelid", Value: bson.D{{Key: "$last", Value: "$modelid"}}},
					{Key: "contentid", Value: bson.D{{Key: "$last", Value: "$contentid"}}},
					{Key: "type", Value: bson.D{{Key: "$last", Value: "$type"}}},
					{Key: "predictions", Value: bson.D{{Key: "$push", Value: "$predictions"}}},
				},
			},
		},
		bson.D{{Key: "$sample", Value: bson.D{{Key: "size", Value: sampleCount}}}},
	}

	cursor, err := collection.Aggregate(
		context.TODO(),
		pipeline,
		&options.AggregateOptions{AllowDiskUse: common.Ptr(true)})
	if err != nil {
		return nil, err
	}

	if err := cursor.All(context.TODO(), &predictions); err != nil {
		return nil, err
	}

	return predictions, nil
}

// Count counts the number of predictions in collection.
func (p Prediction) Count(db *db.DB, model models.Model) (int64, error) {
	collection := db.Client.Database(DATABASE).Collection(PREDICTION_COLLECTION)

	filter := bson.M{
		"userid":  model.UserID,
		"modelid": model.ID.Hex(),
	}

	return collection.CountDocuments(context.TODO(), filter)
}

// Create creates a new Prediction to the db
func (p Prediction) Create(db *db.DB, predictions []models.Prediction) (int, error) {
	collection := db.Client.Database(DATABASE).Collection(PREDICTION_COLLECTION)

	inserts := []mongo.WriteModel{}
	for _, prediction := range predictions {
		inserts = append(inserts, mongo.NewInsertOneModel().SetDocument(prediction))
	}

	result, err := collection.BulkWrite(context.TODO(), inserts)
	if err != nil {
		return 0, err
	}

	if int(result.InsertedCount) != len(predictions) {
		return 0, fmt.Errorf("unexpected number of inserted docuemnts; docuemnts provided=%d; documents inserted=%d", len(predictions), result.InsertedCount)
	}

	return int(result.InsertedCount), nil
}

// DeleteMany deletes multiple Predictions based on a filter
func (p Prediction) DeleteMany(db *db.DB, userid, modelid string) error {
	collection := db.Client.Database(DATABASE).Collection(PREDICTION_COLLECTION)

	filter := bson.M{"userid": userid, "modelid": modelid}

	if _, err := collection.DeleteMany(context.TODO(), filter); err != nil {
		return err
	}

	return nil
}

// Finds and returns all prediction associated with a filter.
func (p Prediction) Query(db *db.DB, userid string, q models.Query) ([]models.Prediction, int64, error) {
	var prediction []models.Prediction

	collection := db.Client.Database(DATABASE).Collection(PREDICTION_COLLECTION)

	options := options.Find()

	// Sort by `_id` field descending (default) or specified sort key and set pagination options
	options.SetSort(bson.M{q.SortKey: q.SortVal})
	options.SetLimit(int64(q.Limit))
	options.SetSkip(int64(q.Offset))

	// Build filter
	filter, err := q.NewQueryFilter(userid)
	if err != nil {
		return nil, 0, err
	}

	cursor, err := collection.Find(context.TODO(), filter, options)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(context.TODO())

	if err := cursor.All(context.TODO(), &prediction); err != nil {
		return nil, 0, err
	}

	count, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, 0, err
	}

	return prediction, count, nil
}

// GetContentPredictions retrieves all predictions for a given contentId, modelId, and predictionId
func (p Prediction) View(db *db.DB, userid, id string) (*models.Prediction, error) {
	collection := db.Client.Database(DATABASE).Collection(PREDICTION_COLLECTION)

	objId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrPredictionDoesNotExist
	}

	filter := bson.M{"_id": objId, "userid": userid}

	prediction := models.Prediction{}

	if err := collection.FindOne(context.TODO(), filter).Decode(&prediction); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrPredictionDoesNotExist
		}
		return nil, err
	}

	return &prediction, nil
}
