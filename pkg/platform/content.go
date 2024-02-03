/*
 * File: content.go
 * Project: platform
 * File Created: Sunday, 9th October 2022 6:44:27 pm
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
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	common "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
)

// Custom errors
var (
	ErrContentDoesNotExist    = echo.NewHTTPError(http.StatusNotFound, "Content does not exists.")
	ErrContentAlreadyExists   = echo.NewHTTPError(http.StatusConflict, "Content already exists.")
	ErrUnexpectedContentCount = echo.NewHTTPError(http.StatusInternalServerError, "Unexpected content count.")
	ErrNullTagMultiFilter     = echo.NewHTTPError(http.StatusBadRequest, "null tag filter can not be specified with other tags")
)

// Content represents the client for content table
type Content struct{}

func NewContent() *Content {
	return &Content{}
}

// ContentDB represents content repository interface
type ContentDB interface {
	Index(*db.DB) error
	Create(*db.DB, *models.Content) (*models.Content, error)
	View(*db.DB, string, string) (*models.Content, error)
	ViewAnnotation(*db.DB, string, string, string) (*models.Content, error)
	Sample(*db.DB, string, string, int, bool) ([]models.Content, error)
	Query(*db.DB, string, models.Query) ([]models.Content, int64, error)
	Update(*db.DB, *models.Content) error
	Delete(*db.DB, string) error
	PullProjectAssociation(*db.DB, string, string) error
	FindOrphanedContent(*db.DB, string, ...*options.FindOptions) (*mongo.Cursor, error)
	FindProjectContent(*db.DB, string, string, ...*options.FindOptions) (*mongo.Cursor, error)
	FindAnnotated(*db.DB, string, string, string, string, models.Pagination, ...string) ([]models.Content, int, error)

	aggregate(*db.DB, interface{}, interface{}, ...*options.AggregateOptions) error
}

func (c Content) Index(db *db.DB) error {
	collection := db.Client.Database(DATABASE).Collection(CONTENT_COLLECTION)

	models := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "userid", Value: 1}, {Key: "projects", Value: 1}},
			Options: &options.IndexOptions{Background: common.Ptr(true)},
		},
		{
			Keys:    bson.D{{Key: "userid", Value: 1}, {Key: "datasetid", Value: 1}, {Key: "tagids.*", Value: 1}},
			Options: &options.IndexOptions{Background: common.Ptr(true)},
		},
		{
			Keys:    bson.D{{Key: "_id", Value: 1}, {Key: "datasetid", Value: 1}},
			Options: &options.IndexOptions{Unique: common.Ptr(true), Background: common.Ptr(true)},
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

func (c Content) FindAnnotated(db *db.DB, userid, projectid, datasetid, operator string, p models.Pagination, tagids ...string) ([]models.Content, int, error) {
	if datasetid != "" {
		return c.findAnnotatedContent(db, userid, datasetid, operator, p, tagids...)
	}
	return c.findUnannotated(db, userid, projectid, p)
}

func (c Content) findUnannotated(db *db.DB, userid, projectid string, p models.Pagination) ([]models.Content, int, error) {
	lookup := bson.M{
		"from":         ANNOTATION_COLLECTION,
		"localField":   "_id",
		"foreignField": "contentid",
		"as":           "matched_annotation",
		"pipeline": bson.A{
			bson.M{"$match": bson.M{"projectid": projectid}},
		},
	}

	pipeline := []bson.M{
		{"$match": bson.M{
			"userid":   userid,
			"projects": projectid,
		}},
		{"$lookup": lookup},
		{"$match": bson.M{"matched_annotation": bson.M{"$eq": bson.A{}}}},
		{"$facet": bson.M{
			"metadata": bson.A{bson.M{"$count": "total"}},
			"content": bson.A{
				bson.M{"$sort": bson.M{p.SortKey: p.SortVal}},
				bson.M{"$skip": p.Offset},
				bson.M{"$limit": p.Limit},
				bson.M{"$lookup": lookup},
			}},
		},
	}

	var results []struct {
		Metadata []struct {
			Total int `bson:"total"`
		} `bson:"metadata"`
		Content []models.Content `bson:"content"`
	}

	allowDiskUse := true
	batchSize := int32(100)
	opts := options.AggregateOptions{AllowDiskUse: &allowDiskUse, BatchSize: &batchSize}
	if err := c.aggregate(db, pipeline, &results, &opts); err != nil {
		return nil, 0, err
	}

	if len(results) > 0 {
		count := 0
		if len(results[0].Metadata) > 0 {
			count = results[0].Metadata[0].Total
		}
		return results[0].Content, count, nil
	}

	return []models.Content{}, 0, nil
}

func (c Content) findAnnotatedContent(db *db.DB, userid, datasetid, operator string, p models.Pagination, tagids ...string) ([]models.Content, int, error) {
	collection := db.Client.Database(DATABASE).Collection(ANNOTATION_COLLECTION)

	// Validate operator
	if operator != "or" && operator != "and" {
		return nil, 0, fmt.Errorf("operation must be one of `or`, `and`")
	}

	// Validate if null is specified, it's the only tag
	for _, tag := range tagids {
		if strings.ToLower(tag) == "null" && len(tagids) > 1 {
			return nil, 0, ErrNullTagMultiFilter
		}
	}

	// Get annotations
	match := bson.M{
		"$and": []interface{}{
			bson.M{"userid": userid},
			bson.M{"datasetid": datasetid},
		},
	}

	andTags, orTags := []string{}, []string{}
	if len(tagids) > 0 {
		for i := 0; i < len(tagids); i++ {
			if strings.ToLower(tagids[i]) == "null" {
				match["$and"] = append(match["$and"].([]interface{}), bson.M{"tagids": bson.M{"$size": 0}})
			} else {
				if operator == "and" {
					andTags = append(andTags, tagids[i])
				} else {
					orTags = append(orTags, tagids[i])
				}
			}
		}
	}

	if len(andTags) > 0 {
		// Results that strictly match the specified tags
		// e.g. specify [Tag-A, Tag-B] and ONLY results with Tag-A AND Tag-B are returned
		match["$and"] = append(match["$and"].([]interface{}), bson.M{"tagids": bson.M{"$eq": andTags}})
	} else if len(orTags) > 0 {
		// Results that contain any of the specified tags
		// e.g. specify [Tag-A, Tag-B] and results with Tag-A OR Tag-B are returned
		match["$and"] = append(match["$and"].([]interface{}), bson.M{"tagids": bson.M{"$in": orTags}})
	}

	// Retrieve associated content
	lookup := bson.M{
		"from":         CONTENT_COLLECTION,
		"localField":   "contentid",
		"foreignField": "_id",
		"as":           "content",
	}

	pipeline := []bson.M{
		{"$match": match},
		{"$sort": bson.M{p.SortKey: p.SortVal}},
		{"$skip": p.Offset},
		{"$limit": p.Limit},
		{"$lookup": lookup},
	}

	var annotations []models.Annotation

	allowDiskUse := true
	batchSize := int32(1000)
	opts := options.AggregateOptions{AllowDiskUse: &allowDiskUse, BatchSize: &batchSize}
	cursor, err := collection.Aggregate(context.TODO(), pipeline, &opts)
	if err != nil {
		return nil, 0, err
	}

	if err = cursor.All(context.TODO(), &annotations); err != nil {
		return nil, 0, err
	}

	// Rearrange
	contentSlice := []models.Content{}
	for _, annotation := range annotations {
		content := annotation.Content[0]
		content.Annotation = []models.Annotation{annotation}
		contentSlice = append(contentSlice, content)
	}

	count, err := collection.CountDocuments(context.TODO(), match)
	if err != nil {
		return nil, 0, err
	}

	return contentSlice, int(count), err
}

// Create creates a new content to the db
func (c Content) Create(db *db.DB, content *models.Content) (*models.Content, error) {
	collection := db.Client.Database(DATABASE).Collection(CONTENT_COLLECTION)

	// Insert content
	if _, err := collection.InsertOne(context.TODO(), content); err != nil {
		return nil, err
	}

	return content, nil
}

func (c Content) Update(db *db.DB, content *models.Content) error {
	collection := db.Client.Database(DATABASE).Collection(CONTENT_COLLECTION)

	filter := bson.M{
		"$and": []interface{}{
			bson.M{"_id": content.ID},
			bson.M{"userid": content.UserID},
		},
	}

	content.UpdatedAt = time.Now()

	update := bson.M{
		"$set": content,
	}

	_, err := collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return err
	}

	return nil
}

// View retrieves content from the database
func (c Content) ViewAnnotation(db *db.DB, userid, contentid, datasetid string) (*models.Content, error) {
	collection := db.Client.Database(DATABASE).Collection(CONTENT_COLLECTION)

	lookup := bson.M{
		"from":         ANNOTATION_COLLECTION,
		"localField":   "_id",
		"foreignField": "contentid",
		"as":           "annotation",
		"pipeline": bson.A{
			bson.M{"$match": bson.M{"datasetid": datasetid}},
		},
	}

	pipeline := []bson.M{
		{"$match": bson.M{
			"_id":    contentid,
			"userid": userid,
		}},
		{"$lookup": lookup},
	}

	var results []models.Content

	allowDiskUse := true
	batchSize := int32(100)
	opts := options.AggregateOptions{AllowDiskUse: &allowDiskUse, BatchSize: &batchSize}
	cursor, err := collection.Aggregate(context.TODO(), pipeline, &opts)
	if err != nil {
		return nil, err
	}

	if err = cursor.All(context.TODO(), &results); err != nil {
		return nil, err
	}

	switch len(results) {
	case 0:
		return nil, ErrContentDoesNotExist
	case 1:
		return &results[0], nil
	default:
		// This should never happend!
		return nil, ErrUnexpectedContentCount
	}
}

// View retrieves content from the database
func (c Content) View(db *db.DB, userid, contentid string) (*models.Content, error) {
	collection := db.Client.Database(DATABASE).Collection(CONTENT_COLLECTION)

	// Check if content exists for user
	filter := bson.M{
		"$and": []interface{}{
			bson.M{"_id": contentid},
			bson.M{"userid": userid},
		},
	}

	content := &models.Content{}

	err := collection.FindOne(context.TODO(), filter).Decode(&content)
	if err := mongoErr(err); err != nil {
		return content, err
	}

	return content, nil
}

// Delete deletes content based on id.
func (c Content) Delete(db *db.DB, id string) error {
	collection := db.Client.Database(DATABASE).Collection(CONTENT_COLLECTION)

	_, err := collection.DeleteOne(context.TODO(), bson.M{"_id": id})
	if err != nil {
		return err
	}

	return nil
}

// Sample retrieves a random sample of content from the database
func (c Content) Sample(db *db.DB, userid, projectid string, count int, filterAnnotated bool) ([]models.Content, error) {
	collection := db.Client.Database(DATABASE).Collection(CONTENT_COLLECTION)

	// ensure we don't accidentally query the entire DB
	if count > 1000 {
		count = 1000
	}

	var pipeline []bson.M
	if !filterAnnotated {
		pipeline = []bson.M{
			{"$match": bson.M{
				"userid":   userid,
				"projects": projectid,
			}},
			{"$sample": bson.M{
				"size": count,
			}},
		}
	} else {
		lookup := bson.M{
			"from":         ANNOTATION_COLLECTION,
			"localField":   "_id",
			"foreignField": "contentid",
			"as":           "matched_annotation",
			"pipeline": bson.A{
				bson.M{"$match": bson.M{"projectid": projectid}},
			},
		}

		pipeline = []bson.M{
			{"$match": bson.M{
				"userid":   userid,
				"projects": projectid,
			}},
			{"$lookup": lookup},
			{"$match": bson.M{"matched_annotation": bson.M{"$eq": bson.A{}}}},
			{"$sample": bson.M{
				"size": count,
			}},
		}
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 20*time.Second)
	defer cancel()

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var content []models.Content

	if err = cursor.All(context.TODO(), &content); err != nil {
		return nil, err
	}

	return content, nil
}

// Finds content no longer associated with any projects.
// It is up to the caller to close the returned cursor.
func (c Content) FindOrphanedContent(db *db.DB, userid string, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	collection := db.Client.Database(DATABASE).Collection(CONTENT_COLLECTION)
	filter := bson.M{
		"$and": []bson.M{
			{"userid": userid},
			{"projects": bson.M{"$size": 0}},
		},
	}

	cursor, err := collection.Find(context.TODO(), filter, opts...)
	if err != nil {
		return nil, err
	}

	return cursor, nil
}

func (c Content) FindProjectContent(db *db.DB, userid, projectid string, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	collection := db.Client.Database(DATABASE).Collection(CONTENT_COLLECTION)
	filter := bson.M{
		"$and": []bson.M{
			{"userid": userid},
			{"projects": projectid},
		},
	}
	cursor, err := collection.Find(context.TODO(), filter, opts...)
	if err != nil {
		return nil, err
	}

	return cursor, nil
}

// Finds and returns all content associated with a filter.
func (c Content) Query(db *db.DB, userid string, q models.Query) ([]models.Content, int64, error) {
	var content []models.Content

	collection := db.Client.Database(DATABASE).Collection(CONTENT_COLLECTION)

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

	if err := cursor.All(context.TODO(), &content); err != nil {
		return nil, 0, err
	}

	count, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, 0, err
	}

	return content, count, nil
}

func (c Content) PullProjectAssociation(db *db.DB, userid, projectid string) error {
	collection := db.Client.Database(DATABASE).Collection(CONTENT_COLLECTION)
	filter := bson.M{
		"$and": []interface{}{
			bson.M{"userid": userid},
			bson.M{"projects": projectid},
		},
	}
	update := bson.M{"$pull": bson.M{"projects": projectid}}
	_, err := collection.UpdateMany(
		context.TODO(),
		filter,
		update,
	)
	return err
}

// aggregate executes an aggregate command and returns a pointer to the resulting documents.
// Passed in results should be a reference type.
func (c Content) aggregate(db *db.DB, pipeline interface{}, results interface{}, opts ...*options.AggregateOptions) error {
	collection := db.Client.Database(DATABASE).Collection(CONTENT_COLLECTION)

	cursor, err := collection.Aggregate(context.TODO(), pipeline, opts...)
	if err != nil {
		return err
	}
	defer cursor.Close(context.TODO())

	return cursor.All(context.TODO(), results)
}

func mongoErr(err error) error {
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in the collection
		if err == mongo.ErrNoDocuments {
			return ErrContentDoesNotExist
		}
		return err
	}
	return nil
}
