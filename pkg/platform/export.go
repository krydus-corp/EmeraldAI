/*
 * File: export.go
 * Project: platform
 * File Created: Tuesday, 26th October 2021 10:34:25 pm
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
	ErrExportDoesNotExist = echo.NewHTTPError(http.StatusNotFound, "Export does not exist.")
)

// Export represents the client for export table
type Export struct{}

func NewExport() *Export {
	return &Export{}
}

// ExportDB represents export repository interface
type ExportDB interface {
	Index(*db.DB) error
	Create(*db.DB, models.Export) (models.Export, error)
	View(*db.DB, string, string) (models.Export, error)
	List(*db.DB, string, models.Pagination) ([]models.Export, int64, error)
	Query(*db.DB, string, models.Query) ([]models.Export, int64, error)
	Update(*db.DB, models.Export) error
	FindProjectExports(*db.DB, string, string, ...*options.FindOptions) (*mongo.Cursor, error)
	Delete(*db.DB, primitive.ObjectID) error
	RemoveModelFromExports(*db.DB, string, string) error
}

func (e *Export) Index(db *db.DB) error {
	collection := db.Client.Database(DATABASE).Collection(EXPORT_COLLECTION)

	models := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "_id", Value: 1}, {Key: "userid", Value: 1}},
			Options: &options.IndexOptions{Unique: common.Ptr(true), Background: common.Ptr(true)},
		},
		{
			Keys:    bson.D{{Key: "userid", Value: 1}, {Key: "projectid", Value: 1}},
			Options: &options.IndexOptions{Background: common.Ptr(true)},
		},
	}

	if _, err := collection.Indexes().CreateMany(context.TODO(), models); err != nil {
		return err
	}
	return nil
}

// Create creates a new export to the db
func (e *Export) Create(db *db.DB, export models.Export) (models.Export, error) {
	collection := db.Client.Database(DATABASE).Collection(EXPORT_COLLECTION)

	// Insert Model
	if _, err := collection.InsertOne(context.TODO(), export); err != nil {
		return models.Export{}, err
	}

	return export, nil
}

func (e *Export) View(db *db.DB, userid, exportid string) (models.Export, error) {
	var export models.Export

	objID, err := primitive.ObjectIDFromHex(exportid)
	if err != nil {
		return models.Export{}, err
	}

	collection := db.Client.Database(DATABASE).Collection(EXPORT_COLLECTION)
	if err = collection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&export); err != nil {
		if err == mongo.ErrNoDocuments {
			return models.Export{}, ErrExportDoesNotExist
		}
		return models.Export{}, err
	}
	return export, nil
}

func (e *Export) List(db *db.DB, userid string, p models.Pagination) ([]models.Export, int64, error) {
	var exports []models.Export

	collection := db.Client.Database(DATABASE).Collection(EXPORT_COLLECTION)

	options := options.Find()

	// Sort by `_id` field descending (default) or specified sort key and set pagination options
	options.SetSort(bson.M{p.SortKey: p.SortVal})
	options.SetLimit(int64(p.Limit))
	options.SetSkip(int64(p.Offset))

	filter := bson.M{"userid": userid}

	cursor, err := collection.Find(context.TODO(), filter, options)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(context.TODO())

	err = cursor.All(context.TODO(), &exports)
	if err != nil {
		return nil, 0, err
	}

	count, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, 0, err
	}

	return exports, count, nil
}

func (e *Export) Update(db *db.DB, export models.Export) error {
	collection := db.Client.Database(DATABASE).Collection(EXPORT_COLLECTION)

	filter := bson.M{
		"$and": []interface{}{
			bson.M{"_id": export.ID},
			bson.M{"userid": export.UserID},
		},
	}

	export.UpdatedAt = time.Now()

	update := bson.M{
		"$set": export,
	}

	_, err := collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return err
	}

	return err
}

func (e *Export) UpdateMany(db *db.DB, filter, update interface{}) error {
	collection := db.Client.Database(DATABASE).Collection(EXPORT_COLLECTION)

	if _, err := collection.UpdateMany(context.TODO(), filter, update); err != nil {
		return err
	}
	return nil
}

func (e *Export) FindProjectExports(db *db.DB, userid, projectid string, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	collection := db.Client.Database(DATABASE).Collection(EXPORT_COLLECTION)

	filter := bson.M{
		"$and": []interface{}{
			bson.M{"projectid": projectid},
			bson.M{"userid": userid},
		},
	}

	cursor, err := collection.Find(context.TODO(), filter, opts...)
	if err != nil {
		return nil, err
	}

	return cursor, nil
}

// Finds and returns all content associated with a filter.
func (e *Export) Query(db *db.DB, userid string, q models.Query) ([]models.Export, int64, error) {
	var exports []models.Export

	collection := db.Client.Database(DATABASE).Collection(EXPORT_COLLECTION)

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

	if err := cursor.All(context.TODO(), &exports); err != nil {
		return nil, 0, err
	}

	count, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, 0, err
	}

	return exports, count, nil
}

// Delete export by _id
func (e *Export) Delete(db *db.DB, exportid primitive.ObjectID) error {
	collection := db.Client.Database(DATABASE).Collection(EXPORT_COLLECTION)

	_, err := collection.DeleteOne(context.TODO(), bson.M{
		"$and": []interface{}{
			bson.M{"_id": exportid},
		},
	})

	return err
}

func (e *Export) RemoveModelFromExports(db *db.DB, userid, modelid string) error {
	filter := bson.M{
		"$and": []interface{}{
			bson.M{"userid": userid},
			bson.M{"modelid": modelid},
		},
	}

	update := bson.M{"$set": bson.M{"modelid": ""}}

	return e.UpdateMany(db, filter, update)
}
