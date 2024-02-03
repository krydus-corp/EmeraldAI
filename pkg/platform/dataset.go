/*
 * File: dataset.go
 * Dataset: platform
 * File Created: Saturday, 30th April 2022 12:53:58 pm
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
	"time"

	"github.com/aws/aws-sdk-go/aws"
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
	ErrDatasetDoesNotExist = echo.NewHTTPError(http.StatusNotFound, "Dataset does not exists.")
	ErrDatasetCorruptState = echo.NewHTTPError(http.StatusInternalServerError, "Project dataset's in corrupt state")
	ErrDatasetLocked       = echo.NewHTTPError(http.StatusConflict, "Dataset locked and not modifiable.")
)

// Dataset represents the client for dataset table
type Dataset struct{}

func NewDataset() *Dataset {
	return &Dataset{}
}

// DatasetDB represents dataset repository interface
type DatasetDB interface {
	Index(*db.DB) error
	Create(*db.DB, *models.Dataset) (*models.Dataset, error)
	View(*db.DB, string, string) (*models.Dataset, error)
	Update(*db.DB, *models.Dataset) error
	Copy(*db.DB, *models.Dataset, bool, *primitive.ObjectID) (*models.Dataset, error)
	FindVersion(*db.DB, string, string, int, ...*options.FindOptions) (*mongo.Cursor, error)
	Delete(*db.DB, primitive.ObjectID) error
	DeleteProjectDatasets(*db.DB, string, string) error
}

func (d Dataset) Index(db *db.DB) error {
	collection := db.Client.Database(DATABASE).Collection(DATASET_COLLECTION)

	models := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "_id", Value: 1}, {Key: "locked", Value: 1}},
			Options: &options.IndexOptions{Unique: common.Ptr(true), Background: common.Ptr(true)},
		},
		{
			Keys:    bson.D{{Key: "userid", Value: 1}, {Key: "projectid", Value: 1}, {Key: "version", Value: 1}},
			Options: &options.IndexOptions{Background: common.Ptr(true)},
		},
		{
			Keys:    bson.D{{Key: "_id", Value: 1}, {Key: "userid", Value: 1}},
			Options: &options.IndexOptions{Unique: common.Ptr(true), Background: common.Ptr(true)},
		},
	}

	if _, err := collection.Indexes().CreateMany(context.TODO(), models); err != nil {
		return err
	}
	return nil
}

func (d Dataset) Copy(db *db.DB, datasetToCopy *models.Dataset, lockDataset bool, newDatasetID *primitive.ObjectID) (*models.Dataset, error) {
	datasetCollection := db.Client.Database(DATABASE).Collection(DATASET_COLLECTION)
	tagCollection := db.Client.Database(DATABASE).Collection(TAG_COLLECTION)
	annotationCollection := db.Client.Database(DATABASE).Collection(ANNOTATION_COLLECTION)
	contentCollection := db.Client.Database(DATABASE).Collection(CONTENT_COLLECTION)

	options := options.FindOptions{AllowDiskUse: aws.Bool(true)}

	// Version count
	count, err := datasetCollection.CountDocuments(context.TODO(), bson.M{"userid": datasetToCopy.UserID, "projectid": datasetToCopy.ProjectID})
	if err != nil {
		return nil, err
	}

	// Create new dataset
	newDataset := models.NewDataset(datasetToCopy.UserID, datasetToCopy.ProjectID)
	newDataset.Version = int(count) // indexed at zero
	if newDatasetID != nil {
		newDataset.ID = *newDatasetID
	}
	if lockDataset {
		newDataset.Locked = true
	}
	newDataset.Split = datasetToCopy.Split
	if _, err := datasetCollection.InsertOne(context.TODO(), newDataset); err != nil {
		return nil, err
	}

	// Copy over tags
	cursor, err := tagCollection.Find(context.TODO(), bson.M{
		"userid":    datasetToCopy.UserID,
		"datasetid": datasetToCopy.ID.Hex(),
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var (
		newTags []interface{}
		tagMap  = make(map[string]string) // old -> new
	)

	for cursor.Next(context.TODO()) {
		var tag models.Tag
		if err = cursor.Decode(&tag); err != nil {
			return nil, fmt.Errorf("error retrieving tag from database; user=%s dataset=%s err=%s", datasetToCopy.UserID, datasetToCopy.ID.Hex(), err.Error())

		}
		t := models.NewTag(tag.UserID, tag.ProjectID, newDataset.ID.Hex(), tag.Name, tag.Property)
		newTags = append(newTags, t)
		tagMap[tag.ID.Hex()] = t.ID.Hex()
	}

	_, err = tagCollection.InsertMany(context.TODO(), newTags)
	if err != nil {
		return nil, err
	}

	// Copy over annotations
	cursor, err = annotationCollection.Find(context.TODO(), bson.M{
		"userid":    datasetToCopy.UserID,
		"datasetid": datasetToCopy.ID.Hex(),
	}, &options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var newAnnotations []interface{}
	for cursor.Next(context.TODO()) {
		var annotation models.Annotation
		if err = cursor.Decode(&annotation); err != nil {
			return nil, fmt.Errorf("error retrieving annotation from database; user=%s dataset=%s err=%s", datasetToCopy.UserID, datasetToCopy.ID.Hex(), err.Error())

		}

		mappedTagIDs := []string{}
		for _, tag := range annotation.TagIDs {
			if t, ok := tagMap[tag]; ok {
				mappedTagIDs = append(mappedTagIDs, t)
			} else {
				return nil, fmt.Errorf("error mapping new tags ids; user=%s dataset=%s err=%s", datasetToCopy.UserID, datasetToCopy.ID.Hex(), err.Error())
			}
		}
		mappedMetadata := models.AnnotationMetadata{BoundingBoxes: []models.AnnotationDataBoundingBox{}}
		for _, box := range annotation.Metadata.BoundingBoxes {
			if t, ok := tagMap[box.TagID]; ok {
				newBox := box
				newBox.TagID = t
				mappedMetadata.BoundingBoxes = append(mappedMetadata.BoundingBoxes, newBox)
			} else {
				return nil, fmt.Errorf("error mapping new tags ids; user=%s dataset=%s err=%s", datasetToCopy.UserID, datasetToCopy.ID.Hex(), err.Error())
			}
		}

		t := models.NewAnnotation(annotation.UserID, annotation.ProjectID, newDataset.ID.Hex(), annotation.ContentID, mappedTagIDs, annotation.Base64Image, mappedMetadata, annotation.ContentMetadata)

		newAnnotations = append(newAnnotations, t)
	}

	_, err = annotationCollection.InsertMany(context.TODO(), newAnnotations)
	if err != nil {
		return nil, err
	}

	// Copy over dataset associations
	if _, err := contentCollection.UpdateMany(context.TODO(),
		bson.M{"userid": newDataset.UserID, "projects": newDataset.ProjectID},
		bson.M{"$addToSet": bson.M{"datasets": newDataset.ID.Hex()}}); err != nil {
		return nil, err
	}

	return newDataset, nil
}

// Create creates a new dataset to the db
func (d Dataset) Create(db *db.DB, dataset *models.Dataset) (*models.Dataset, error) {
	collection := db.Client.Database(DATABASE).Collection(DATASET_COLLECTION)

	if err := dataset.Valid(); err != nil {
		return nil, err
	}

	// Insert Dataset
	if _, err := collection.InsertOne(context.TODO(), dataset); err != nil {
		return nil, err
	}

	return dataset, nil
}

func (d Dataset) View(db *db.DB, userid, datasetid string) (*models.Dataset, error) {
	collection := db.Client.Database(DATABASE).Collection(DATASET_COLLECTION)

	datasetidPrimitive, err := primitive.ObjectIDFromHex(datasetid)
	if err != nil {
		return nil, ErrDatasetDoesNotExist
	}

	// Check existing Dataset
	filter := bson.M{
		"$and": []interface{}{
			bson.M{"_id": datasetidPrimitive},
			bson.M{"userid": userid},
		},
	}
	dataset := &models.Dataset{}

	err = collection.FindOne(context.TODO(), filter).Decode(&dataset)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in the collection
		if err == mongo.ErrNoDocuments {
			return dataset, ErrDatasetDoesNotExist
		}
		return dataset, err
	}
	return dataset, nil
}

// Updates updates a dataset's fields.
func (d Dataset) Update(db *db.DB, dataset *models.Dataset) error {
	collection := db.Client.Database(DATABASE).Collection(DATASET_COLLECTION)

	// Check if locked
	if dataset.Locked {
		return ErrDatasetLocked
	}

	var update = make(map[string]interface{})
	update["updated_at"] = time.Now()
	if !dataset.IsZeroValue() {
		if err := dataset.Valid(); err != nil {
			return err
		}
		update["split"] = dataset.Split
	}

	filter := bson.M{
		"$and": []interface{}{
			bson.M{"_id": dataset.ID},
			bson.M{"userid": dataset.UserID},
		},
	}

	_, err := collection.UpdateOne(context.TODO(), filter, bson.M{"$set": update})
	if err != nil {
		return err
	}

	return nil
}

func (d Dataset) List(db *db.DB, userid string, page models.Pagination) ([]models.Dataset, error) {
	var datasets []models.Dataset

	collection := db.Client.Database(DATABASE).Collection(DATASET_COLLECTION)

	options := options.Find()

	// Sort by `_id` field descending (default) or specified sort key and set pagination options
	options.SetSort(bson.M{page.SortKey: page.SortVal})
	options.SetLimit(int64(page.Limit))
	options.SetSkip(int64(page.Offset))

	cursor, err := collection.Find(context.TODO(), bson.M{"userid": userid}, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	err = cursor.All(context.TODO(), &datasets)
	if err != nil {
		return nil, err
	}

	return datasets, nil
}

// Finds and returns all datasets associated with a filter.
func (d Dataset) Query(db *db.DB, userid string, q models.Query) ([]models.Dataset, error) {
	var datasets []models.Dataset

	collection := db.Client.Database(DATABASE).Collection(DATASET_COLLECTION)

	options := options.Find()

	// Sort by `_id` field descending (default) or specified sort key and set pagination options
	options.SetSort(bson.M{q.SortKey: q.SortVal})
	options.SetLimit(int64(q.Limit))
	options.SetSkip(int64(q.Offset))

	// Build filter
	filter, err := q.NewQueryFilter(userid)
	if err != nil {
		return nil, err
	}

	cursor, err := collection.Find(context.TODO(), filter, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	if err := cursor.All(context.TODO(), &datasets); err != nil {
		return nil, err
	}

	return datasets, nil
}

func (d *Dataset) FindVersion(db *db.DB, userid, projectid string, version int, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	collection := db.Client.Database(DATABASE).Collection(DATASET_COLLECTION)

	filter := bson.M{"userid": userid, "projectid": projectid, "version": version}

	cursor, err := collection.Find(context.TODO(), filter, opts...)
	if err != nil {
		return nil, err
	}

	return cursor, nil
}

// Delete deletes a dataset based on id.
func (d Dataset) Delete(db *db.DB, id primitive.ObjectID) error {
	collection := db.Client.Database(DATABASE).Collection(DATASET_COLLECTION)
	_, err := collection.DeleteOne(context.TODO(), bson.M{"_id": id})
	if err != nil {
		return err
	}

	return nil
}

// DeleteProjectDatasets deletes datasets for a project.
func (d Dataset) DeleteProjectDatasets(db *db.DB, userid, projectid string) error {
	collection := db.Client.Database(DATABASE).Collection(DATASET_COLLECTION)
	filter := bson.M{
		"$and": []interface{}{
			bson.M{"projectid": projectid},
			bson.M{"userid": userid},
		},
	}
	_, err := collection.DeleteMany(context.TODO(), filter)
	if err != nil {
		return err
	}

	return nil
}
