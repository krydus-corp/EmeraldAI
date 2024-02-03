/*
 * File: annotation.go
 * Project: platform
 * File Created: Monday, 2nd May 2022 7:42:31 pm
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
	ErrAnnotationAlreadyExists = echo.NewHTTPError(http.StatusConflict, "Annotation already exists.")
	ErrAnnotationDoesNotExist  = echo.NewHTTPError(http.StatusNotFound, "Annotation does not exists.")
)

// Annotation represents the client for annotation table
type Annotation struct{}

func NewAnnotation() *Annotation {
	return &Annotation{}
}

// AnnotationDB represents annotation repository interface
type AnnotationDB interface {
	Index(*db.DB) error
	Create(*db.DB, models.Annotation) (*models.Annotation, error)
	View(*db.DB, string, string) (*models.Annotation, error)
	List(*db.DB, string, string, string, models.Pagination) ([]models.Annotation, int64, error)
	Query(*db.DB, string, models.Query) ([]models.Annotation, int64, error)
	Update(*db.DB, *models.Annotation) error
	DeleteTagID(*db.DB, string, string, string, string) error

	AnnotationsAssociatedWithContent(*db.DB, string, string, string) ([]string, error)
	TotalAnnotations(*db.DB, string, string, string, interface{}) error
	AverageAnnotationsPerImage(*db.DB, string, string, string, interface{}) error
	AnnotationsPerClass(*db.DB, string, string, string, interface{}) error
	AnnotationsImageInsights(*db.DB, string, string, string, interface{}) error
	AnnotationsImageStat(*db.DB, string, string, string, string, interface{}) error
	CountNullAnnotations(*db.DB, string, string, string) (*int64, error)
	CountAnnotations(*db.DB, string, string) (*int64, error)
	CountUnannotated(*db.DB, string, string, string) (*int64, error)
	FindContentAnnotation(*db.DB, string, string, string, string) (*models.Annotation, error)
	FindDatasetAnnotations(*db.DB, string, string, ...*options.FindOptions) (*mongo.Cursor, error)
	DeleteUserAnnotations(*db.DB, string, []primitive.ObjectID) error
	DeleteProjectAnnotations(*db.DB, string, string) error
	DeleteDatasetAnnotations(*db.DB, string, string, string) error

	aggregate(*db.DB, interface{}, interface{}) error
}

// Finds annotations associated with a dataset.
// It is up to the caller to close the returned cursor.
func (a Annotation) FindDatasetAnnotations(db *db.DB, userid, datasetid string, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	collection := db.Client.Database(DATABASE).Collection(ANNOTATION_COLLECTION)

	filter := bson.M{
		"userid":    userid,
		"datasetid": datasetid,
	}

	cursor, err := collection.Find(context.TODO(), filter, opts...)
	if err != nil {
		return nil, err
	}

	return cursor, nil
}

func (a Annotation) Index(db *db.DB) error {
	collection := db.Client.Database(DATABASE).Collection(ANNOTATION_COLLECTION)

	models := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "userid", Value: 1}, {Key: "projectid", Value: 1}, {Key: "datasetid", Value: 1}},
			Options: &options.IndexOptions{Background: common.Ptr(true)},
		},
		{
			Keys:    bson.D{{Key: "userid", Value: 1}, {Key: "datasetid", Value: 1}, {Key: "projectid", Value: 1}, {Key: "contentid", Value: 1}},
			Options: &options.IndexOptions{Background: common.Ptr(true)},
		},
		{
			Keys:    bson.D{{Key: "userid", Value: 1}, {Key: "datasetid", Value: 1}, {Key: "projectid", Value: 1}, {Key: "tagids", Value: 1}},
			Options: &options.IndexOptions{Background: common.Ptr(true)},
		},
		{
			Keys:    bson.D{{Key: "userid", Value: 1}, {Key: "projectid", Value: 1}, {Key: "contentid", Value: 1}},
			Options: &options.IndexOptions{Background: common.Ptr(true)},
		},
		{
			Keys:    bson.D{{Key: "datasetid", Value: 1}, {Key: "projectid", Value: 1}},
			Options: &options.IndexOptions{Background: common.Ptr(true)},
		},
		{
			Keys:    bson.D{{Key: "_id", Value: 1}, {Key: "userid", Value: 1}},
			Options: &options.IndexOptions{Unique: common.Ptr(true), Background: common.Ptr(true)},
		},
		{
			Keys:    bson.D{{Key: "contentid", Value: 1}, {Key: "projectid", Value: 1}},
			Options: &options.IndexOptions{Background: common.Ptr(true)},
		},
	}

	if _, err := collection.Indexes().CreateMany(context.TODO(), models); err != nil {
		return err
	}
	return nil
}

// Create creates a new annotation to the db
func (a Annotation) Create(db *db.DB, annotation models.Annotation) (*models.Annotation, error) {
	collection := db.Client.Database(DATABASE).Collection(ANNOTATION_COLLECTION)
	contentDB := NewContent()

	// Check existing Annotation
	filter := bson.M{
		"$and": []interface{}{
			bson.M{"userid": annotation.UserID},
			bson.M{"datasetid": annotation.DatasetID},
			bson.M{"projectid": annotation.ProjectID},
			bson.M{"contentid": annotation.ContentID},
		},
	}

	// Insert Annotation
	result, err := collection.UpdateOne(context.TODO(), filter, bson.M{"$setOnInsert": annotation}, &options.UpdateOptions{Upsert: common.Ptr(true)})
	if err != nil {
		return &models.Annotation{}, err
	}

	if result.MatchedCount > 0 {
		return &models.Annotation{}, ErrAnnotationAlreadyExists
	}

	// Update content with associated dataset
	content, err := contentDB.View(db, annotation.UserID, annotation.ContentID)
	if err != nil {
		return nil, err
	}

	err = contentDB.Update(db, content)
	if err != nil {
		return nil, err
	}

	return &annotation, nil
}

// View returns single annotation by ID
func (a Annotation) View(db *db.DB, userid, id string) (*models.Annotation, error) {
	var annotation models.Annotation

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	collection := db.Client.Database(DATABASE).Collection(ANNOTATION_COLLECTION)
	if err = collection.FindOne(context.TODO(), bson.M{"_id": objID, "userid": userid}).Decode(&annotation); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrAnnotationDoesNotExist
		}
		return nil, err
	}
	return &annotation, nil
}

// List returns list of all annotations.
func (a Annotation) List(db *db.DB, userid, projectid, datasetid string, p models.Pagination) ([]models.Annotation, int64, error) {
	var annotations = []models.Annotation{}

	collection := db.Client.Database(DATABASE).Collection(ANNOTATION_COLLECTION)

	options := options.Find()

	// Sort by `_id` field descending (default) or specified sort key and set pagination options
	options.SetSort(bson.M{p.SortKey: p.SortVal})
	options.SetLimit(int64(p.Limit))
	options.SetSkip(int64(p.Offset))

	filter := bson.M{
		"userid":    userid,
		"datasetid": datasetid,
		"projectid": projectid}

	cursor, err := collection.Find(context.TODO(), filter, options)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(context.TODO())

	err = cursor.All(context.TODO(), &annotations)
	if err != nil {
		return nil, 0, err
	}

	count, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, 0, err
	}

	return annotations, count, nil
}

// Finds and returns all annotation associated with a filter.
func (a Annotation) Query(db *db.DB, userid string, q models.Query) ([]models.Annotation, int64, error) {
	var annotation []models.Annotation

	collection := db.Client.Database(DATABASE).Collection(ANNOTATION_COLLECTION)

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

	if err := cursor.All(context.TODO(), &annotation); err != nil {
		return nil, 0, err
	}

	count, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, 0, err
	}

	return annotation, count, nil
}

func (a Annotation) Update(db *db.DB, annotation *models.Annotation) error {
	collection := db.Client.Database(DATABASE).Collection(ANNOTATION_COLLECTION)

	filter := bson.M{
		"$and": []interface{}{
			bson.M{"_id": annotation.ID},
			bson.M{"userid": annotation.UserID},
		},
	}

	var update = make(map[string]interface{})
	update["updated_at"] = time.Now()

	if annotation.Split != "" {
		update["split"] = annotation.Split
	}

	if len(annotation.Metadata.BoundingBoxes) != 0 {
		update["metadata"] = annotation.Metadata
	}
	if annotation.Base64Image != "" {
		update["b64_image"] = annotation.Base64Image
	}

	switch {
	// No update
	case annotation.TagIDs == nil:
		break
		// Null annotation
	case len(annotation.TagIDs) == 0:
		update["null_annotation"] = true
		update["tagids"] = []string{}
		update["metadata"] = models.AnnotationMetadata{}
		// Update
	case len(annotation.TagIDs) > 0:
		update["null_annotation"] = false
		update["tagids"] = annotation.TagIDs
	}

	_, err := collection.UpdateOne(
		context.TODO(),
		filter,
		bson.M{
			"$set": update,
		})

	return err
}

func (a Annotation) DeleteTagID(db *db.DB, userid, projectid, datasetid, tagid string) error {
	collection := db.Client.Database(DATABASE).Collection(ANNOTATION_COLLECTION)

	// Pull tagid from annotations
	updateModel := mongo.NewUpdateManyModel().SetFilter(bson.M{
		"$and": []interface{}{
			bson.M{"userid": userid},
			bson.M{"datasetid": datasetid},
			bson.M{"projectid": projectid},
		},
	}).SetUpdate(bson.M{
		"$pull": bson.M{
			"metadata.bounding_boxes": bson.M{"tagid": tagid},
			"tagids":                  tagid,
		},
	})

	// Delete any annotations that are not null annotations and have no tag associations
	deleteModel := mongo.NewDeleteManyModel().SetFilter(bson.M{
		"$and": []interface{}{
			bson.M{"userid": userid},
			bson.M{"datasetid": datasetid},
			bson.M{"projectid": projectid},
			bson.M{"tagids": bson.M{"$size": 0}},
			bson.M{"null_annotation": false},
		}})

	models := []mongo.WriteModel{updateModel, deleteModel}
	opts := options.BulkWrite().SetOrdered(true)

	_, err := collection.BulkWrite(context.TODO(), models, opts)

	return err
}

func (a Annotation) AnnotationsAssociatedWithContent(db *db.DB, userid, projectid, contentid string) (annotationIDs []string, err error) {
	lookup := bson.M{
		"from": "dataset",
		"let":  bson.M{"did": "$datasetid"},
		"pipeline": []bson.M{
			{"$match": bson.M{
				"$expr": bson.M{
					"$eq": bson.A{"$_id", bson.M{"$toObjectId": "$$did"}},
				},
				"locked": false,
			}},
		},
		"as": "dataset_doc",
	}

	pipeline := []bson.M{
		{"$match": bson.M{
			"userid":    userid,
			"projectid": projectid,
			"contentid": contentid,
		}},
		{"$lookup": lookup},
		{"$unwind": "$dataset_doc"},
		{"$project": bson.M{
			"_id": 1,
		}},
	}

	var results []struct {
		ID string `json:"id" bson:"_id"`
	}

	if err := a.aggregate(db, pipeline, &results); err != nil {
		return nil, err
	}

	for _, result := range results {
		annotationIDs = append(annotationIDs, result.ID)
	}

	return
}

func (a Annotation) TotalAnnotations(db *db.DB, userid, projectid, datasetid string, results interface{}) error {
	pipeline := []bson.M{
		{"$match": bson.M{
			"userid":    userid,
			"datasetid": datasetid,
			"projectid": projectid,
		}},
		{"$group": bson.M{"_id": nil, "null_count": bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{0, bson.M{"$size": "$tagids"}}}, 1, 0}}}, "count": bson.M{"$sum": bson.M{"$size": "$tagids"}}}},
	}

	return a.aggregate(db, pipeline, results)
}

// TODO - for bounding box, this should average the number of boxes per image
func (a Annotation) AverageAnnotationsPerImage(db *db.DB, userid, projectid, datasetid string, results interface{}) error {
	pipeline := []bson.M{
		{"$match": bson.M{
			"userid":    userid,
			"datasetid": datasetid,
			"projectid": projectid,
		}},
		{"$project": bson.M{
			"annotation_sizes": bson.M{
				"$size": "$tagids",
			},
		}},
		{"$group": bson.M{
			"_id": nil,
			"average_annotations": bson.M{
				"$avg": "$annotation_sizes",
			},
		}},
	}

	return a.aggregate(db, pipeline, results)
}

func (a Annotation) MaxBoundingBoxesPerImage(db *db.DB, userid, projectid, datasetid string, results interface{}) error {
	pipeline := []bson.M{
		{"$match": bson.M{
			"userid":                  userid,
			"datasetid":               datasetid,
			"projectid":               projectid,
			"metadata.bounding_boxes": bson.M{"$exists": true},
		}},
		{"$project": bson.M{
			"bounding_boxes": bson.M{
				"$size": "$metadata.bounding_boxes",
			},
		}},
		{"$group": bson.M{
			"_id": nil,
			"max_bounding_boxes": bson.M{
				"$max": "$bounding_boxes",
			},
		}},
	}

	return a.aggregate(db, pipeline, results)
}

func (a Annotation) AnnotationsPerClass(db *db.DB, userid, projectid, datasetid string, results interface{}) error {
	pipeline := []bson.M{
		{"$match": bson.M{
			"userid":    userid,
			"datasetid": datasetid,
			"projectid": projectid,
		}},
		{"$unwind": "$tagids"},
		{"$group": bson.M{"_id": "$tagids", "count": bson.M{"$sum": 1}}},
	}

	return a.aggregate(db, pipeline, results)
}

func (a Annotation) AnnotationsImageInsights(db *db.DB, userid, projectid, datasetid string, results interface{}) error {
	// Dimensions
	lookup := bson.M{
		"from":         CONTENT_COLLECTION,
		"localField":   "contentid",
		"foreignField": "_id",
		"as":           "content",
	}

	pipeline := []bson.M{
		{"$match": bson.M{
			"userid":    userid,
			"datasetid": datasetid,
			"projectid": projectid,
		}},
		{"$lookup": lookup},
		{"$unwind": "$content"},
		{"$project": bson.M{
			"_id":    0,
			"height": "$content.height",
			"width":  "$content.width",
		}},
	}

	return a.aggregate(db, pipeline, results)
}

func (a Annotation) AnnotationsImageStat(db *db.DB, userid, datasetid, stat, field string, results interface{}) error {
	if stat != "avg" && stat != "min" && stat != "max" {
		return fmt.Errorf("image stat must be one of: avg, min, or max")
	}

	if field != "size" && field != "height" && field != "width" {
		return fmt.Errorf("image stat field must be one of: size, height, or width")
	}

	pipeline := []bson.M{
		{"$match": bson.M{
			"userid":    userid,
			"datasetid": datasetid,
		}},
		{"$group": bson.M{
			"_id": nil,
			"stat": bson.M{
				fmt.Sprintf("$%s", stat): fmt.Sprintf("$content_metadata.%s", field),
			},
		}},
	}

	return a.aggregate(db, pipeline, results)
}

func (a Annotation) CountNullAnnotations(db *db.DB, userid, projectid, datasetid string) (*int64, error) {
	collection := db.Client.Database(DATABASE).Collection(ANNOTATION_COLLECTION)

	count, err := collection.CountDocuments(context.TODO(), bson.M{
		"userid":    userid,
		"datasetid": datasetid,
		"projectid": projectid,
		"tagids":    bson.M{"$size": 0},
	})
	return common.Ptr(count), err
}

func (a Annotation) CountAnnotations(db *db.DB, userid, datasetid string) (*int64, error) {
	collection := db.Client.Database(DATABASE).Collection(ANNOTATION_COLLECTION)

	count, err := collection.CountDocuments(context.TODO(), bson.M{
		"$and": []interface{}{
			bson.M{"userid": userid},
			bson.M{"datasetid": datasetid},
		},
	})
	return common.Ptr(count), err
}

func (a Annotation) CountUnannotated(db *db.DB, userid, datasetid, projectid string) (*int64, error) {
	collection := db.Client.Database(DATABASE).Collection(ANNOTATION_COLLECTION)

	annotatedContentCount, err := collection.CountDocuments(context.TODO(), bson.M{
		"$and": []interface{}{
			bson.M{"userid": userid},
			bson.M{"datasetid": datasetid},
		},
	})
	if err != nil {
		return nil, err
	}

	collection = db.Client.Database(DATABASE).Collection(CONTENT_COLLECTION)
	totalContentCount, err := collection.CountDocuments(context.TODO(), bson.M{
		"$and": []interface{}{
			bson.M{"userid": userid},
			bson.M{"projects": projectid},
		},
	})
	return common.Ptr(totalContentCount - annotatedContentCount), err
}

func (a Annotation) FindContentAnnotation(db *db.DB, userid, projectid, datasetid, contentid string) (*models.Annotation, error) {
	collection := db.Client.Database(DATABASE).Collection(ANNOTATION_COLLECTION)

	filter := bson.M{
		"userid":    userid,
		"datasetid": datasetid,
		"projectid": projectid,
		"contentid": contentid,
	}
	annotation := models.Annotation{}

	err := collection.FindOne(context.TODO(), filter).Decode(&annotation)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrAnnotationDoesNotExist
		}
		return nil, err
	}

	return &annotation, nil
}

func (a Annotation) DeleteUserAnnotations(db *db.DB, userid string, annotationids []primitive.ObjectID) error {
	collection := db.Client.Database(DATABASE).Collection(ANNOTATION_COLLECTION)

	_, err := collection.DeleteMany(context.TODO(), bson.M{
		"$and": []interface{}{
			bson.M{"_id": bson.M{"$in": annotationids}},
			bson.M{"userid": userid},
		}})

	return err
}

func (a Annotation) DeleteProjectAnnotations(db *db.DB, userid, projectid string) error {
	collection := db.Client.Database(DATABASE).Collection(ANNOTATION_COLLECTION)

	_, err := collection.DeleteMany(context.TODO(), bson.M{
		"$and": []interface{}{
			bson.M{"userid": userid},
			bson.M{"projectid": projectid},
		}})

	return err
}

// Delete all annotations for associated dataset
func (a Annotation) DeleteDatasetAnnotations(db *db.DB, userid, projectid, datasetid string) error {
	collection := db.Client.Database(DATABASE).Collection(ANNOTATION_COLLECTION)

	_, err := collection.DeleteMany(context.TODO(), bson.M{
		"$and": []interface{}{
			bson.M{"userid": userid},
			bson.M{"projectid": projectid},
			bson.M{"datasetid": datasetid},
		},
	})

	return err
}

// aggregate executes an aggregate command and returns a pointer to the resulting documents.
// Passed in results should be a reference type.
func (a Annotation) aggregate(db *db.DB, pipeline interface{}, results interface{}) error {
	collection := db.Client.Database(DATABASE).Collection(ANNOTATION_COLLECTION)

	cursor, err := collection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(context.TODO())

	return cursor.All(context.TODO(), results)
}
