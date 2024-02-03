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

// Project represents the client for project table
type Project struct{}

func NewProject() *Project {
	return &Project{}
}

// Custom errors
var (
	ErrProjectAlreadyExists = echo.NewHTTPError(http.StatusConflict, "Project already exists.")
	ErrProjectDoesNotExist  = echo.NewHTTPError(http.StatusNotFound, "Project does not exists.")
)

// ProjectDB represents project repository interface
type ProjectDB interface {
	Create(*db.DB, models.Project) (models.Project, error)
	View(*db.DB, string, string) (models.Project, error)
	ViewByName(*db.DB, string, string) (models.Project, error)
	Update(*db.DB, models.Project) error
	List(*db.DB, string, models.Pagination) ([]models.Project, int64, error)
	Query(*db.DB, string, models.Query) ([]models.Project, int64, error)
	Delete(*db.DB, string, string) error
}

// Create is a method for creating a new project to the db.
func (p Project) Create(db *db.DB, project models.Project) (models.Project, error) {
	collection := db.Client.Database(DATABASE).Collection(PROJECT_COLLECTION)

	// Check existing Project i.e. same name
	filter := bson.M{
		"$and": []interface{}{
			bson.M{"userid": project.UserID},
			bson.M{"name": project.Name},
		},
	}

	// Insert Project
	result, err := collection.UpdateOne(context.TODO(), filter, bson.M{"$setOnInsert": project}, &options.UpdateOptions{Upsert: common.Ptr(true)})
	if err != nil {
		return models.Project{}, err
	}

	if result.MatchedCount > 0 {
		return models.Project{}, ErrProjectAlreadyExists
	}

	return project, nil
}

// View is a method for viewing a project by ID.
func (p Project) View(db *db.DB, userid, projectid string) (models.Project, error) {
	collection := db.Client.Database(DATABASE).Collection(PROJECT_COLLECTION)

	projectidPrimitive, err := primitive.ObjectIDFromHex(projectid)
	if err != nil {
		return models.Project{}, ErrProjectDoesNotExist
	}

	// Check existing Project
	filter := bson.M{
		"$and": []interface{}{
			bson.M{"userid": userid},
			bson.M{"_id": projectidPrimitive},
		},
	}
	project := models.Project{}

	err = collection.FindOne(context.TODO(), filter).Decode(&project)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in the collection
		if err == mongo.ErrNoDocuments {
			return project, ErrProjectDoesNotExist
		}
		return project, err
	}
	return project, nil
}

// ViewByName is a method for viewing a project by name.
func (p Project) ViewByName(db *db.DB, userid, projectName string) (models.Project, error) {
	collection := db.Client.Database(DATABASE).Collection(PROJECT_COLLECTION)

	// Check existing Project
	filter := bson.M{
		"$and": []interface{}{
			bson.M{"userid": userid},
			bson.M{"name": projectName},
		},
	}
	project := models.Project{}

	err := collection.FindOne(context.TODO(), filter).Decode(&project)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in the collection
		if err == mongo.ErrNoDocuments {
			return project, ErrProjectDoesNotExist
		}
		return project, err
	}
	return project, nil
}

// Update is a method for updating a project's fields.
// If updating count, use the UpdateCount Method to ensure atomicity.
func (p Project) Update(db *db.DB, project models.Project) error {
	collection := db.Client.Database(DATABASE).Collection(PROJECT_COLLECTION)

	// Update
	var update = make(map[string]interface{})

	update["updated_at"] = time.Now()
	if project.Name != "" {
		// If project name is changing, check if it is available
		if err := projectExists(collection, project); err != nil {
			return err
		}
		update["name"] = project.Name
	}
	if project.Description != nil {
		update["description"] = project.Description
	}
	if project.LicenseType != nil {
		update["license"] = project.LicenseType
	}
	if project.Profile100 != "" {
		update["profile_100"] = project.Profile100
	}
	if project.Profile200 != "" {
		update["profile_200"] = project.Profile200
	}
	if project.Profile640 != "" {
		update["profile_640"] = project.Profile640
	}

	filter := bson.M{
		"$and": []interface{}{
			bson.M{"userid": project.UserID},
			bson.M{"_id": project.ID},
		},
	}

	if _, err := collection.UpdateOne(
		context.TODO(),
		filter,
		bson.M{"$set": update}); err != nil {
		return err
	}

	return nil
}

// Updates updates a project's count atomically.
func (p Project) UpdateCount(db *db.DB, projectid, userid string, i int) error {
	collection := db.Client.Database(DATABASE).Collection(PROJECT_COLLECTION)

	projectidPrimitive, err := primitive.ObjectIDFromHex(projectid)
	if err != nil {
		return ErrProjectDoesNotExist
	}

	filter := bson.M{
		"$and": []interface{}{
			bson.M{"userid": userid},
			bson.M{"_id": projectidPrimitive},
		},
	}

	update := bson.M{"$inc": bson.M{"count": i}}

	_, err = collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return err
	}

	return nil
}

// List is a method for listing out projects.
func (p Project) List(db *db.DB, userid string, page models.Pagination) ([]models.Project, int64, error) {
	var projects []models.Project

	collection := db.Client.Database(DATABASE).Collection(PROJECT_COLLECTION)

	options := options.Find()

	// Sort by `_id` field descending (default) or specified sort key and set pagination options
	options.SetSort(bson.M{page.SortKey: page.SortVal})
	options.SetLimit(int64(page.Limit))
	options.SetSkip(int64(page.Offset))

	filter := bson.M{"userid": userid}
	cursor, err := collection.Find(context.TODO(), filter, options)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(context.TODO())

	err = cursor.All(context.TODO(), &projects)
	if err != nil {
		return nil, 0, err
	}

	count, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, 0, err
	}

	return projects, count, nil
}

// Delete is a method for deleting a project by ID.
func (p Project) Delete(db *db.DB, userid, projectid string) error {
	// Check if the project exists
	_, err := p.View(db, userid, projectid)
	if err != nil && err == ErrProjectDoesNotExist {
		return ErrProjectDoesNotExist
	}

	// Delete Project
	collection := db.Client.Database(DATABASE).Collection(PROJECT_COLLECTION)
	projectidPrimitive, err := primitive.ObjectIDFromHex(projectid)
	if err != nil {
		return ErrProjectDoesNotExist
	}

	filter := bson.M{
		"$and": []interface{}{
			bson.M{"userid": userid},
			bson.M{"_id": projectidPrimitive},
		},
	}

	_, err = collection.DeleteOne(context.TODO(), filter)
	if err != nil {
		return err
	}

	return nil
}

// Query is a method for finding and returning all projects associated with a filter.
func (p Project) Query(db *db.DB, userid string, q models.Query) ([]models.Project, int64, error) {
	var projects []models.Project

	collection := db.Client.Database(DATABASE).Collection(PROJECT_COLLECTION)

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

	if err := cursor.All(context.TODO(), &projects); err != nil {
		return nil, 0, err
	}

	count, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, 0, err
	}

	return projects, count, nil
}

// Count counts the number of projects in collection.
func (p Project) Count(db *db.DB, filter interface{}) (int64, error) {
	collection := db.Client.Database(DATABASE).Collection(PROJECT_COLLECTION)
	return collection.CountDocuments(context.TODO(), filter)
}

// projectExists checks if project name exists in collection for a project other than itself.
func projectExists(collection *mongo.Collection, project models.Project) error {
	filter := bson.M{
		"$and": []interface{}{
			bson.M{"userid": project.UserID},
			bson.M{"name": project.Name},
			bson.M{"_id": bson.M{"$ne": project.ID}},
		},
	}

	count, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrProjectAlreadyExists
	}
	return nil
}
