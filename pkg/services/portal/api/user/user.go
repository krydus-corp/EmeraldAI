// Package user contains user application services
package user

import (
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"

	common "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	image "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/image"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	mail "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/mail"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
)

const (
	defaultUserGroup = "user"
)

var (
	ErrInvalidEmailVerificationCode = echo.NewHTTPError(400, "invalid email verification code")
	ErrInvalidImageType             = echo.NewHTTPError(400, "uploaded content must be an image (.jpeg, .jpg, or .png)")
)

// Create creates a new user account
func (u User) Create(c echo.Context, req models.User) (models.User, error) {
	req.Password = u.sec.Hash(req.Password)

	// Database operation
	user, err := u.platform.UserDB.Create(u.db, req)
	if err != nil {
		return models.User{}, err
	}

	// Add RBAC policy
	_, err = u.enforcer.AddGroupingPolicy(user.Username, defaultUserGroup)
	if err != nil {
		return models.User{}, err
	}
	if err := u.enforcer.SavePolicy(); err != nil {
		return models.User{}, err
	}

	return user, nil
}

// List returns list of users
func (u User) List(c echo.Context, p models.Pagination) ([]models.User, int64, error) {
	return u.platform.UserDB.List(u.db, p)
}

// View returns single user
func (u User) View(c echo.Context, id string) (models.User, error) {
	return u.platform.UserDB.View(u.db, id)
}

// Delete deletes a user
func (u User) Delete(c echo.Context, id string) error {
	user, err := u.platform.UserDB.Delete(u.db, id)
	if err != nil {
		return err
	}

	// Remove RBAC policy
	_, err = u.enforcer.RemoveGroupingPolicy(user.Username, defaultUserGroup)
	if err != nil {
		return err
	}

	// Save new policy
	if err := u.enforcer.SavePolicy(); err != nil {
		return err
	}

	return nil
}

// Update contains user's information used for updating
type Update struct {
	ID                    string
	FirstName             string
	LastName              string
	Email                 string
	EmailVerificationCode string
	IsAdmin               bool
}

// Update updates user's contact information
// !! When adding additional user updates, remember to propagate them to the DB layer !!
func (u User) Update(c echo.Context, r Update) (models.User, error) {
	id, err := primitive.ObjectIDFromHex(r.ID)
	if err != nil {
		return models.User{}, err
	}

	// If update includes an email update and not an admin, check email verification code
	if !r.IsAdmin && r.Email != "" {
		// Check reset token
		key := fmt.Sprintf("%s-%d", r.Email, mail.MailConfirmation)
		token, err := u.cache.Get(key)
		if err != nil {
			log.Errorf("email verification token not found; key=%s", key)
			return models.User{}, ErrInvalidEmailVerificationCode
		}

		if token != strings.TrimSpace(r.EmailVerificationCode) {
			log.Errorf("email verification code mismatch; provided=%s expected=%s", r.EmailVerificationCode, token)
			return models.User{}, ErrInvalidEmailVerificationCode
		}
	}

	if err := u.platform.UserDB.UpdateContact(u.db, models.User{
		ID:        id,
		FirstName: r.FirstName,
		LastName:  r.LastName,
		Email:     r.Email,
	}); err != nil {
		return models.User{}, err
	}

	return u.platform.UserDB.View(u.db, r.ID)
}

func (u User) Profile(c echo.Context, userid, filename string, file multipart.File) error {
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	// Create base64 images
	stats, err := image.GetStats(fileBytes)
	if err != nil {
		return err
	}

	if !strings.Contains(stats.ContentType, "jpg") && !strings.Contains(stats.ContentType, "jpeg") && !strings.Contains(stats.ContentType, "png") {
		return ErrInvalidImageType
	}

	// Create thumbnail
	thumb100, _, err := image.Thumbnail(fileBytes, 100, 100)
	if err != nil {
		return errors.Wrapf(err, "unable to create thumbnail from image; file=%s", filename)
	}

	thumb200, _, err := image.Thumbnail(fileBytes, 200, 200)
	if err != nil {
		return errors.Wrapf(err, "unable to create thumbnail from image; file=%s", filename)
	}

	thumb400, _, err := image.Thumbnail(fileBytes, 400, 400)
	if err != nil {
		return errors.Wrapf(err, "unable to create thumbnail from image; file=%s", filename)
	}

	id, err := primitive.ObjectIDFromHex(userid)
	if err != nil {
		return err
	}

	return u.platform.UserDB.UpdateProfile(u.db, models.User{
		ID:         id,
		Profile100: &thumb100,
		Profile200: &thumb200,
		Profile400: &thumb400,
	})
}

func (u User) DeleteProfile(c echo.Context, userid string) error {
	id, err := primitive.ObjectIDFromHex(userid)
	if err != nil {
		return err
	}

	return u.platform.UserDB.UpdateProfile(u.db, models.User{
		ID:         id,
		Profile100: common.Ptr(""),
		Profile200: common.Ptr(""),
		Profile400: common.Ptr(""),
	})
}

func (u User) EmailVerificationCode(c echo.Context, userid, email string) error {
	user, err := u.platform.UserDB.View(u.db, userid)
	if err != nil {
		return err
	}

	// Send verification mail
	mailType := mail.MailConfirmation
	mailData := &mail.PortalMailData{
		Username: user.Username,
		Code:     common.GenerateRandomString(8),
	}

	mailReq := u.mail.NewMail(u.mail.PassResetEmail, []string{email}, u.mail.PassResetSubject, mailType, mailData)
	err = u.mail.SendMail(mailReq)
	if err != nil {
		log.Errorf("unable to send mail", "error", err)
		return err
	}

	// cache the password reset code
	key, val := fmt.Sprintf("%s-%d", email, mail.MailConfirmation), mailData.Code
	exp := time.Minute * time.Duration(u.mail.PassResetCodeExpiration)
	if err := u.cache.Set(key, val, exp); err != nil {
		return err
	}

	return nil
}

func (u User) EmailVerificationCodeConfirm(c echo.Context, email, token string) error {
	// Check reset token
	key := fmt.Sprintf("%s-%d", email, mail.MailConfirmation)
	cachedToken, err := u.cache.Get(key)
	if err != nil {
		log.Errorf("email verification token not found; key=%s", key)
		return ErrInvalidEmailVerificationCode
	}

	if cachedToken != strings.TrimSpace(token) {
		log.Errorf("email verification code mismatch; provided=%s expected=%s", token, cachedToken)
		return ErrInvalidEmailVerificationCode
	}

	return nil
}
