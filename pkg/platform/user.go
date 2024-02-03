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

// User represents the client for user table
type User struct{}

func NewUser() *User {
	return &User{}
}

// Custom errors
var (
	ErrUserDoesNotExist  = echo.NewHTTPError(http.StatusNotFound, "User does not exist.")
	ErrUserAlreadyExists = echo.NewHTTPError(http.StatusConflict, "Username or email already exists.")
)

// UserDB represents user repository interface
type UserDB interface {
	Index(*db.DB) error
	View(*db.DB, string) (models.User, error)
	FindByUsername(*db.DB, string) (models.User, error)
	UpdatePassword(*db.DB, models.User) error
	UpdateContact(*db.DB, models.User) error
	Create(*db.DB, models.User) (models.User, error)
	List(*db.DB, models.Pagination) ([]models.User, int64, error)
	Delete(*db.DB, string) (models.User, error)

	AddUsage(*db.DB, string, models.Usage) error
}

func (u User) Index(db *db.DB) error {
	collection := db.Client.Database(DATABASE).Collection(USER_COLLECTION)

	models := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: &options.IndexOptions{Unique: common.Ptr(true), Background: common.Ptr(true)},
		},
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: &options.IndexOptions{Unique: common.Ptr(true), Background: common.Ptr(true)},
		},
	}

	if _, err := collection.Indexes().CreateMany(context.TODO(), models); err != nil {
		return err
	}
	return nil
}

// View returns single user by ID
func (u User) View(db *db.DB, id string) (models.User, error) {
	var user models.User

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return models.User{}, ErrUserDoesNotExist
	}

	collection := db.Client.Database(DATABASE).Collection(USER_COLLECTION)
	if err = collection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&user); err != nil {
		if err == mongo.ErrNoDocuments {
			return models.User{}, ErrUserDoesNotExist
		}
		return models.User{}, err
	}
	return user, nil
}

// FindByUsername queries for single user by username
func (u User) FindByUsername(db *db.DB, uname string) (models.User, error) {
	var user models.User

	filter := bson.M{"username": uname}
	collection := db.Client.Database(DATABASE).Collection(USER_COLLECTION)
	if err := collection.FindOne(context.TODO(), filter).Decode(&user); err != nil {
		if err == mongo.ErrNoDocuments {
			return models.User{}, ErrUserDoesNotExist
		}
		return models.User{}, err
	}

	return user, nil
}

// FindByEmail queries for single user by email
func (u User) FindByEmail(db *db.DB, uname string) (models.User, error) {
	var user models.User

	filter := bson.M{"email": uname}
	collection := db.Client.Database(DATABASE).Collection(USER_COLLECTION)
	if err := collection.FindOne(context.TODO(), filter).Decode(&user); err != nil {
		if err == mongo.ErrNoDocuments {
			return models.User{}, ErrUserDoesNotExist
		}
		return models.User{}, err
	}

	return user, nil
}

// Update updates user's token info
func (u User) UpdateLogin(db *db.DB, user models.User) error {
	collection := db.Client.Database(DATABASE).Collection(USER_COLLECTION)
	_, err := collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{
			"last_login": user.LastLogin,
		}},
	)
	return err
}

// Update updates user's password info
func (u User) UpdatePassword(db *db.DB, user models.User) error {
	collection := db.Client.Database(DATABASE).Collection(USER_COLLECTION)

	_, err := collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{
			"password":             user.Password,
			"last_password_change": user.LastPasswordChange,
		}},
	)
	if err != nil {
		return err
	}

	return nil
}

// Add usage to user account
func (u User) AddUsage(db *db.DB, userid string, usage models.Usage) error {
	collection := db.Client.Database(DATABASE).Collection(USER_COLLECTION)

	user, err := u.View(db, userid)
	if err != nil {
		return err
	}

	user.Billing.AddUsage(usage)

	_, err = collection.UpdateOne(
		context.Background(),
		bson.M{"_id": user.ID},
		bson.M{"$set": user},
	)

	return err
}

// Create creates a new user on database
func (u User) Create(db *db.DB, usr models.User) (models.User, error) {
	collection := db.Client.Database(DATABASE).Collection(USER_COLLECTION)

	// Check existing User
	filter := bson.M{
		"$or": []interface{}{
			bson.M{"username": usr.Username},
			bson.M{"email": usr.Email},
		},
	}
	count, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return models.User{}, err
	}
	if count > 0 {
		return models.User{}, ErrUserAlreadyExists
	}

	// Insert User
	if _, err := collection.InsertOne(context.TODO(), usr); err != nil {
		return models.User{}, err
	}

	return usr, nil
}

// Delete deletes a single user by ID
func (u User) Delete(db *db.DB, id string) (models.User, error) {
	user, err := u.View(db, id)
	if err != nil {
		return models.User{}, err
	}

	collection := db.Client.Database(DATABASE).Collection(USER_COLLECTION)

	_, err = collection.DeleteOne(context.TODO(), bson.M{"_id": user.ID})
	if err != nil {
		return models.User{}, err
	}

	return user, nil
}

// UpdateContact updates user's contact info
func (u User) UpdateContact(db *db.DB, user models.User) error {
	collection := db.Client.Database(DATABASE).Collection(USER_COLLECTION)

	var update = make(map[string]interface{})
	update["updated_at"] = time.Now()
	if user.FirstName != "" {
		update["first_name"] = user.FirstName
	}
	if user.LastName != "" {
		update["last_name"] = user.LastName
	}
	if user.Email != "" {
		update["email"] = user.Email
	}

	_, err := collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": user.ID},
		bson.M{"$set": update})

	return err
}

// UpdateProfile updates user's contact info
func (u User) UpdateProfile(db *db.DB, user models.User) error {
	collection := db.Client.Database(DATABASE).Collection(USER_COLLECTION)

	var update = make(map[string]interface{})
	update["updated_at"] = time.Now()
	if user.Profile100 != nil {
		update["profile_100"] = user.Profile100
	}
	if user.Profile200 != nil {
		update["profile_200"] = user.Profile200
	}
	if user.Profile400 != nil {
		update["profile_400"] = user.Profile400
	}

	_, err := collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": user.ID},
		bson.M{"$set": update})

	return err
}

// List returns list of all users.
func (u User) List(db *db.DB, p models.Pagination) ([]models.User, int64, error) {
	var users []models.User

	collection := db.Client.Database(DATABASE).Collection(USER_COLLECTION)

	options := options.Find()

	// Sort by `_id` field descending (default) or specified sort key and set pagination options
	options.SetSort(bson.M{p.SortKey: p.SortVal})
	options.SetLimit(int64(p.Limit))
	options.SetSkip(int64(p.Offset))

	cursor, err := collection.Find(context.TODO(), bson.M{}, options)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(context.TODO())

	err = cursor.All(context.TODO(), &users)
	if err != nil {
		return nil, 0, err
	}

	count, err := collection.CountDocuments(context.TODO(), bson.M{})
	if err != nil {
		return nil, 0, err
	}

	return users, count, nil
}
