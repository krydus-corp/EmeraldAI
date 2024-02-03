/*
 * File: model.go
 * Project: platform
 * File Created: Thursday, 22nd July 2021 6:10:55 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package platform

import (
	"context"
	"net/http"
	"time"

	"github.com/google/go-cmp/cmp"
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
	ErrModelAlreadyExists     = echo.NewHTTPError(http.StatusConflict, "Model already exists.")
	ErrModelDoesNotExist      = echo.NewHTTPError(http.StatusNotFound, "Model does not exist.")
	ErrModelNameAlreadyExists = echo.NewHTTPError(http.StatusConflict, "Model name already exists.")
)

// Model represents the client for model table
type Model struct{}

func NewModel() *Model {
	return &Model{}
}

// ModelDB represents model repository interface
type ModelDB interface {
	Index(*db.DB) error
	Create(*db.DB, models.Model) (models.Model, error)
	View(*db.DB, string, string) (models.Model, error)
	List(*db.DB, string, string, models.Pagination) ([]models.Model, int64, error)
	Query(*db.DB, string, models.Query) ([]models.Model, int64, error)
	Update(*db.DB, models.Model) error
	DeleteMany(*db.DB, string, primitive.ObjectID) error
	FindProjectModels(*db.DB, string, string, ...*options.FindOptions) (*mongo.Cursor, error)
	ProjectCount(*db.DB, string, string) (int64, error)
	UpdateEndpointStatus(*db.DB, string) error
	AWSEndpointExists(*db.DB, string) (int64, error)
	AWSModelExists(*db.DB, string) (int64, error)
}

func (m Model) Index(db *db.DB) error {
	collection := db.Client.Database(DATABASE).Collection(MODEL_COLLECTION)

	models := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "_id", Value: 1}, {Key: "userid", Value: 1}, {Key: "name", Value: 1}},
			Options: &options.IndexOptions{Unique: common.Ptr(true), Background: common.Ptr(true)},
		},
		{
			Keys:    bson.D{{Key: "userid", Value: 1}, {Key: "projectid", Value: 1}, {Key: "datasetid", Value: 1}},
			Options: &options.IndexOptions{Background: common.Ptr(true)},
		},
	}

	if _, err := collection.Indexes().CreateMany(context.TODO(), models); err != nil {
		return err
	}
	return nil
}

// Create creates a new model to the db
func (m Model) Create(db *db.DB, model models.Model) (models.Model, error) {
	collection := db.Client.Database(DATABASE).Collection(MODEL_COLLECTION)

	// Check existing Model
	// Models and datasets have a 1-1 relationship
	filter := bson.M{
		"$and": []interface{}{
			bson.M{"userid": model.UserID},
			bson.M{"projectid": model.ProjectID},
			bson.M{"datasetid": model.DatasetID},
		},
	}

	count, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return models.Model{}, err
	}
	if count > 0 {
		return models.Model{}, ErrModelAlreadyExists
	}

	// Insert Model
	if _, err := collection.InsertOne(context.TODO(), model); err != nil {
		return models.Model{}, err
	}

	return model, nil
}

func (m Model) View(db *db.DB, userid, modelid string) (models.Model, error) {
	collection := db.Client.Database(DATABASE).Collection(MODEL_COLLECTION)

	modelidStr, err := primitive.ObjectIDFromHex(modelid)
	if err != nil {
		return models.Model{}, ErrModelDoesNotExist
	}

	filter := bson.M{
		"$and": []interface{}{
			bson.M{"_id": modelidStr},
			bson.M{"userid": userid},
		},
	}
	model := models.Model{}

	err = collection.FindOne(context.TODO(), filter).Decode(&model)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in the collection
		if err == mongo.ErrNoDocuments {
			return model, ErrModelDoesNotExist
		}
		return model, err
	}
	return model, nil
}

func (m Model) Update(db *db.DB, model models.Model) error {
	collection := db.Client.Database(DATABASE).Collection(MODEL_COLLECTION)

	filter := bson.M{
		"$and": []interface{}{
			bson.M{"_id": model.ID},
			bson.M{"userid": model.UserID},
		},
	}

	var update = make(map[string]interface{})

	update["updated_at"] = time.Now()
	if model.Name != "" {
		// Check that model name is not already being used.
		if err := modelExists(collection, model); err != nil {
			return err
		}
		update["name"] = model.Name
	}
	if model.TrainingJobName != "" {
		update["training_job_name"] = model.TrainingJobName
	}
	if model.DatasetID != "" {
		update["datasetid"] = model.DatasetID
	}
	if model.Path != "" {
		update["path"] = model.Path
	}
	if model.Metrics != nil {
		update["metrics"] = model.Metrics
	}
	if model.State != "" {
		update["state"] = model.State
	}
	if !time.Time.IsZero(model.TrainStartedAt) {
		update["train_started_at"] = model.TrainStartedAt
	}
	if !time.Time.IsZero(model.TrainEndedAt) {
		update["train_ended_at"] = model.TrainEndedAt
	}
	if !time.Time.IsZero(model.ErrorAt) {
		update["error_at"] = model.ErrorAt
	}
	if model.LastError != nil {
		update["error"] = model.LastError
	}
	if len(model.IntegerMapping) != 0 {
		update["integer_mapping"] = model.IntegerMapping
	}
	if !cmp.Equal(model.Parameters, models.TrainParameters{}) {
		update["parameters"] = model.Parameters
	}
	if !cmp.Equal(model.Augmentation, models.Augmentations{}) {
		update["augmentation"] = model.Augmentation
	}
	if !cmp.Equal(model.Preprocessing, models.Preprocessors{}) {
		update["preprocessing"] = model.Preprocessing
	}
	if !cmp.Equal(model.Deployment, models.Deployment{}) {
		update["deployment"] = model.Deployment
	}
	if !cmp.Equal(model.Batch, models.Batch{}) {
		update["batch"] = model.Batch
	}

	_, err := collection.UpdateOne(
		context.TODO(),
		filter,
		bson.M{"$set": update},
	)

	return err
}

// Delete is a method for deleting a model by ID
func (m Model) Delete(db *db.DB, userid, modelid string) error {
	// Check if the model exists
	_, err := m.View(db, userid, modelid)
	if err != nil && err == ErrModelDoesNotExist {
		return ErrModelDoesNotExist
	}

	// Delete Model
	collection := db.Client.Database(DATABASE).Collection(MODEL_COLLECTION)
	modelidStr, err := primitive.ObjectIDFromHex(modelid)
	if err != nil {
		return ErrModelDoesNotExist
	}

	filter := bson.M{
		"$and": []interface{}{
			bson.M{"_id": modelidStr},
			bson.M{"userid": userid},
		},
	}

	if _, err := collection.DeleteOne(context.TODO(), filter); err != nil {
		return err
	}

	return nil
}

// DeleteMany deletes a model
func (m Model) DeleteMany(db *db.DB, userid string, modelid primitive.ObjectID) error {
	collection := db.Client.Database(DATABASE).Collection(MODEL_COLLECTION)

	_, err := collection.DeleteMany(context.TODO(), bson.M{"_id": modelid, "userid": userid})
	if err != nil {
		return err
	}

	return nil
}

func (m Model) List(db *db.DB, userid, projectid string, p models.Pagination) ([]models.Model, int64, error) {
	var models []models.Model

	collection := db.Client.Database(DATABASE).Collection(MODEL_COLLECTION)

	options := options.Find()

	// Sort by `_id` field descending (default) or specified sort key and set pagination options
	options.SetSort(bson.M{p.SortKey: p.SortVal})
	options.SetLimit(int64(p.Limit))
	options.SetSkip(int64(p.Offset))

	filter := bson.M{
		"userid":    userid,
		"projectid": projectid,
	}

	cursor, err := collection.Find(context.TODO(), filter, options)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(context.TODO())

	err = cursor.All(context.TODO(), &models)
	if err != nil {
		return nil, 0, err
	}

	count, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, 0, err
	}

	return models, count, nil
}

// Finds models associated with a project.
// It is up to the caller to close the returned cursor.
func (m Model) FindProjectModels(db *db.DB, userid, projectid string, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	collection := db.Client.Database(DATABASE).Collection(MODEL_COLLECTION)

	filter := bson.M{
		"$and": []bson.M{
			{"userid": userid},
			{"projectid": projectid},
		},
	}

	cursor, err := collection.Find(context.TODO(), filter, opts...)
	if err != nil {
		return nil, err
	}

	return cursor, nil
}

// Finds and returns all models associated with a filter.
func (m Model) Query(db *db.DB, userid string, q models.Query) ([]models.Model, int64, error) {
	var models []models.Model

	collection := db.Client.Database(DATABASE).Collection(MODEL_COLLECTION)

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

	if err := cursor.All(context.TODO(), &models); err != nil {
		return nil, 0, err
	}

	count, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, 0, err
	}

	return models, count, nil
}

// modelExists checks if model name exists in collection.
func modelExists(collection *mongo.Collection, model models.Model) error {
	filter := bson.M{
		"$and": []interface{}{
			bson.M{"_id": bson.M{"$ne": model.ID}},
			bson.M{"userid": model.UserID},
			bson.M{"projectid": model.ProjectID},
			bson.M{"name": model.Name},
		},
	}

	count, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrModelNameAlreadyExists
	}
	return nil
}

func (m Model) ProjectCount(db *db.DB, userid, projectid string) (int64, error) {
	collection := db.Client.Database(DATABASE).Collection(MODEL_COLLECTION)

	filter := bson.M{
		"$and": []interface{}{
			bson.M{"userid": userid},
			bson.M{"projectid": projectid},
		},
	}
	return collection.CountDocuments(context.TODO(), filter)
}

// Update model endpoint status.
func (m Model) UpdateEndpointStatus(db *db.DB, endpointName string) error {
	collection := db.Client.Database(DATABASE).Collection(MODEL_COLLECTION)

	filter := bson.M{"deployment.endpoint_name": endpointName}

	var update = make(map[string]interface{})

	update["deployment.status"] = "DELETED"

	_, err := collection.UpdateOne(
		context.Background(),
		filter,
		bson.M{"$set": update},
	)

	return err
}

func (m Model) AWSEndpointExists(db *db.DB, endpointName string) (int64, error) {
	collection := db.Client.Database(DATABASE).Collection(MODEL_COLLECTION)
	filter := bson.M{"deployment.endpoint_name": endpointName}
	return collection.CountDocuments(context.TODO(), filter)
}

func (m Model) AWSModelExists(db *db.DB, modelName string) (int64, error) {
	collection := db.Client.Database(DATABASE).Collection(MODEL_COLLECTION)
	filter := bson.M{"deployment.model_name": modelName}
	return collection.CountDocuments(context.TODO(), filter)
}
