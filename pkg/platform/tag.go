package platform

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
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

// Tag represents the client for tag table
type Tag struct{}

func NewTag() *Tag {
	return &Tag{}
}

// Custom errors
var (
	ErrTagAlreadyExists = echo.NewHTTPError(http.StatusConflict, "Tag already exists.")
	ErrTagDoesNotExist  = echo.NewHTTPError(http.StatusNotFound, "Tag does not exists.")
)

// TagDB represents tag repository interface
type TagDB interface {
	Index(*db.DB) error
	Create(*db.DB, models.Tag) (models.Tag, error)
	Update(*db.DB, models.Tag) error
	View(*db.DB, string, string) (models.Tag, error)
	List(*db.DB, string, string, models.Pagination) ([]models.Tag, int64, error)
	ListAll(*db.DB, string, string, ...*options.FindOptions) ([]models.Tag, error)
	Query(*db.DB, string, models.Query) ([]models.Tag, int64, error)
	Find(*db.DB, string, string, ...*options.FindOptions) (*mongo.Cursor, error)
	FindByName(*db.DB, string, string, string, ...*options.FindOneOptions) (models.Tag, error)
	Distinct(*db.DB, string, string, string) ([]interface{}, error)
	DeleteInProject(*db.DB, string, string) error
	DeleteDatasetTags(*db.DB, string, string, string) error
	Delete(*db.DB, primitive.ObjectID) error
	CopyToNewDataset(*db.DB, string, string) error
	TagIntegerMap(*db.DB, string, string) (map[string]int, error)
}

func (t *Tag) Index(db *db.DB) error {
	collection := db.Client.Database(DATABASE).Collection(TAG_COLLECTION)

	models := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "userid", Value: 1}, {Key: "datasetid", Value: 1}, {Key: "name", Value: 1}},
			Options: &options.IndexOptions{Unique: common.Ptr(true), Background: common.Ptr(true)},
		},
		{
			Keys:    bson.D{{Key: "_id", Value: 1}, {Key: "userid", Value: 1}},
			Options: &options.IndexOptions{Unique: common.Ptr(true), Background: common.Ptr(true)},
		},
		{
			Keys:    bson.D{{Key: "userid", Value: 1}, {Key: "datasetid", Value: 1}},
			Options: &options.IndexOptions{Background: common.Ptr(true)},
		},
	}

	if _, err := collection.Indexes().CreateMany(context.TODO(), models); err != nil {
		return err
	}
	return nil
}

func (t *Tag) CopyToNewDataset(db *db.DB, userid, datasetid string) error {
	collection := db.Client.Database(DATABASE).Collection(TAG_COLLECTION)

	filter := bson.M{
		"userid":    userid,
		"datasetid": datasetid,
	}

	cursor, err := collection.Find(context.TODO(), filter)
	if err != nil {
		return err
	}
	defer cursor.Close(context.TODO())

	var newTags []interface{}
	for cursor.Next(context.TODO()) {
		var tag models.Tag
		if err = cursor.Decode(&tag); err != nil {
			return fmt.Errorf("error retrieving tag metadata from database; user=%s dataset=%s err=%s", userid, datasetid, err.Error())

		}
		t := models.NewTag(tag.UserID, tag.ProjectID, tag.DatasetID, tag.Name, tag.Property)
		newTags = append(newTags, t)
	}

	_, err = collection.InsertMany(context.TODO(), newTags)

	return err
}

func (t *Tag) TagIntegerMap(db *db.DB, userid, datasetid string) (map[string]int, error) {
	labelIntegerMap := make(map[string]int)

	tags, err := t.Distinct(db, userid, datasetid, "name")
	if err != nil {
		return nil, err
	}

	for idx, tag := range tags {
		if t, ok := tag.(string); !ok {
			return nil, fmt.Errorf("unexpected type returned during tag lookup; expected=string, returned=%s", reflect.TypeOf(tag).String())
		} else {
			labelIntegerMap[t] = idx
		}
	}

	return labelIntegerMap, nil
}

func (t Tag) Distinct(db *db.DB, userid, datasetid, field string) ([]interface{}, error) {
	collection := db.Client.Database(DATABASE).Collection(TAG_COLLECTION)
	return collection.Distinct(context.TODO(), field, bson.M{"userid": userid, "datasetid": datasetid})
}

// Create creates a new tag to the db
func (t Tag) Create(db *db.DB, tag models.Tag) (models.Tag, error) {
	collection := db.Client.Database(DATABASE).Collection(TAG_COLLECTION)

	tag.Name = strings.TrimSpace(tag.Name)

	// Check existing Tag
	filter := bson.M{
		"$and": []interface{}{
			bson.M{"userid": tag.UserID},
			bson.M{"datasetid": tag.DatasetID},
			bson.M{"name": tag.Name},
		},
	}

	// Insert Tag
	result, err := collection.UpdateOne(context.TODO(), filter, bson.M{"$setOnInsert": tag}, &options.UpdateOptions{Upsert: common.Ptr(true)})
	if err != nil {
		return models.Tag{}, err
	}

	if result.MatchedCount > 0 {
		return models.Tag{}, ErrTagAlreadyExists
	}

	return tag, nil
}

func (t Tag) View(db *db.DB, userid, tagid string) (models.Tag, error) {
	collection := db.Client.Database(DATABASE).Collection(TAG_COLLECTION)

	tagidStr, err := primitive.ObjectIDFromHex(tagid)
	if err != nil {
		return models.Tag{}, ErrTagDoesNotExist
	}

	filter := bson.M{
		"$and": []interface{}{
			bson.M{"userid": userid},
			bson.M{"_id": tagidStr},
		},
	}
	tag := models.Tag{}

	err = collection.FindOne(context.TODO(), filter).Decode(&tag)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in the collection
		if err == mongo.ErrNoDocuments {
			return tag, ErrTagDoesNotExist
		}
		return tag, err
	}
	return tag, nil
}

// Find is a method for viewing tags associated with a userid and projectid.
// It is up to the caller to close the returned cursor.
func (t Tag) Find(db *db.DB, userid, projectid string, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	collection := db.Client.Database(DATABASE).Collection(TAG_COLLECTION)

	cursor, err := collection.Find(context.TODO(), bson.M{
		"$and": []bson.M{
			{"userid": userid},
			{"projectid": projectid},
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	return cursor, nil
}

func (t Tag) FindByName(db *db.DB, userid, datasetid, name string, opts ...*options.FindOneOptions) (models.Tag, error) {
	collection := db.Client.Database(DATABASE).Collection(TAG_COLLECTION)

	tag := models.Tag{}

	err := collection.FindOne(context.TODO(), bson.M{
		"userid":    userid,
		"datasetid": datasetid,
		"name":      name}, opts...).Decode(&tag)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in the collection
		if err == mongo.ErrNoDocuments {
			return tag, ErrTagDoesNotExist
		}
		return tag, err
	}
	return tag, nil
}

// Update updates tag info
func (t Tag) Update(db *db.DB, tag models.Tag) error {
	collection := db.Client.Database(DATABASE).Collection(TAG_COLLECTION)

	filter := bson.M{
		"$and": []interface{}{
			bson.M{"userid": tag.UserID},
			bson.M{"_id": tag.ID},
		},
	}

	var update = make(map[string]interface{})
	update["updated_at"] = time.Now()
	if tag.Name != "" {
		update["name"] = tag.Name
	}
	if len(tag.Property) > 0 {
		update["property"] = tag.Property
	}

	_, err := collection.UpdateOne(
		context.TODO(),
		filter,
		bson.M{"$set": update},
	)

	return err
}

func (t Tag) UpdateCount(db *db.DB, tagid primitive.ObjectID, userid string, i int) error {
	collection := db.Client.Database(DATABASE).Collection(TAG_COLLECTION)

	filter := bson.M{
		"$and": []interface{}{
			bson.M{"userid": userid},
			bson.M{"_id": tagid},
		},
	}

	update := bson.M{"$inc": bson.M{"counts.human": i}}

	_, err := collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return err
	}

	return nil
}

func (t Tag) List(db *db.DB, userid, datasetid string, p models.Pagination) ([]models.Tag, int64, error) {
	var tags []models.Tag

	collection := db.Client.Database(DATABASE).Collection(TAG_COLLECTION)

	options := options.Find()

	// Sort by `_id` field descending (default) or specified sort key and set pagination options
	options.SetSort(bson.M{p.SortKey: p.SortVal})
	options.SetLimit(int64(p.Limit))
	options.SetSkip(int64(p.Offset))

	filter := bson.M{
		"userid":    userid,
		"datasetid": datasetid,
	}

	cursor, err := collection.Find(context.TODO(), filter, options)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(context.TODO())

	err = cursor.All(context.TODO(), &tags)
	if err != nil {
		return nil, 0, err
	}

	count, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, 0, err
	}

	return tags, count, nil
}

func (t Tag) ListAll(db *db.DB, userid, datasetid string, options ...*options.FindOptions) ([]models.Tag, error) {
	var tags []models.Tag

	collection := db.Client.Database(DATABASE).Collection(TAG_COLLECTION)

	cursor, err := collection.Find(context.TODO(), bson.M{
		"userid":    userid,
		"datasetid": datasetid,
	}, options...)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	err = cursor.All(context.TODO(), &tags)
	if err != nil {
		return nil, err
	}

	return tags, nil
}

// Finds and returns all tags associated with a filter.
func (t Tag) Query(db *db.DB, userid string, q models.Query) ([]models.Tag, int64, error) {
	var tags []models.Tag

	collection := db.Client.Database(DATABASE).Collection(TAG_COLLECTION)

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

	if err := cursor.All(context.TODO(), &tags); err != nil {
		return nil, 0, err
	}

	count, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, 0, err
	}

	return tags, count, nil
}

// DeleteInProject deletes tags for a given project.
func (t Tag) DeleteInProject(db *db.DB, userid, projectid string) error {
	collection := db.Client.Database(DATABASE).Collection(TAG_COLLECTION)

	filter := bson.M{
		"$and": []interface{}{
			bson.M{"projectid": projectid},
			bson.M{"userid": userid},
		},
	}

	if _, err := collection.DeleteMany(context.TODO(), filter); err != nil {
		return err
	}

	return nil
}

// Delete all tags of associated dataset.
func (t Tag) DeleteDatasetTags(db *db.DB, userid, projectid, datasetid string) error {
	collection := db.Client.Database(DATABASE).Collection(TAG_COLLECTION)

	filter := bson.M{
		"$and": []interface{}{
			bson.M{"userid": userid},
			bson.M{"projectid": projectid},
			bson.M{"datasetid": datasetid},
		},
	}

	if _, err := collection.DeleteMany(context.TODO(), filter); err != nil {
		return err
	}

	return nil
}

// Delete deletes a tag by id.
func (t Tag) Delete(db *db.DB, id primitive.ObjectID) error {
	collection := db.Client.Database(DATABASE).Collection(TAG_COLLECTION)

	if _, err := collection.DeleteOne(context.TODO(), bson.M{"_id": id}); err != nil {
		return err
	}

	return nil
}
