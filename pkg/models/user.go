package models

import (
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents user domain model
//
// swagger:model User
type User struct {
	// ID for the User
	//
	// swagger:strfmt bsonobjectid
	ID primitive.ObjectID `json:"id" bson:"_id"`
	// Username associated with User
	//
	Username string `json:"username" bson:"username"`
	// User first name
	//
	FirstName string `json:"first_name" bson:"first_name"`
	// User last name
	//
	LastName string `json:"last_name" bson:"last_name"`
	// User email
	//
	Email string `json:"email" bson:"email"`
	// User password
	//
	Password string `json:"-" bson:"password"`

	// User profile picture (100x100 px)
	//
	Profile100 *string `json:"profile_100" bson:"profile_100"`
	// User profile picture (200x200 px)
	//
	Profile200 *string `json:"profile_200" bson:"profile_200"`
	// User profile picture (400x400 px)
	//
	Profile400 *string `json:"profile_400" bson:"profile_400"`

	// User API Key
	//
	APIKey string `json:"api_key" bson:"api_key"`

	// Billing Data
	Billing Billing `json:"-" bson:"billing"`

	CreatedAt          time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" bson:"updated_at"`
	LastLogin          time.Time `json:"last_login,omitempty" bson:"last_login"`
	LastPasswordChange time.Time `json:"last_password_change,omitempty" bson:"last_password_change"`
}

func NewUser(username, password, email, firstName, lastName string) User {
	return User{
		ID:        primitive.NewObjectID(),
		Username:  username,
		Password:  password,
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,

		APIKey: uuid.New().String(),

		Billing: NewBilling(),

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ChangePassword updates user's password related fields
func (u *User) ChangePassword(hash string) {
	u.Password = hash
	u.LastPasswordChange = time.Now()
}
