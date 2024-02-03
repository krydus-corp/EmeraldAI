package user

import (
	"fmt"
	"net/http"

	errs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/error"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"

	"github.com/labstack/echo/v4"
)

const (
	AdminRole = "admin"
)

// HTTP represents user http service
type HTTP struct {
	svc Service
}

// NewHTTP creates new user http service
func NewHTTP(svc Service, r *echo.Group) {
	h := HTTP{svc}
	ur := r.Group("/users")

	//! --------- ADMIN ROUTES ---------- !//
	// swagger:operation POST /v1/users users createUserReq
	// ---
	// summary: Creates new user account.
	// description: |
	//   Creates new user account.
	//   Only accessible by Admin roles.
	// security:
	// - Bearer: []
	// consumes:
	//  - application/json
	// produces:
	//  - application/json
	// responses:
	//   "200":
	//     "schema":
	//      "$ref": "#/definitions/User"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.POST("", h.create)

	// swagger:operation GET /v1/users users listUserReq
	// ---
	// summary: Returns list of users.
	// description: |
	//   Returns list of users.
	//   Only accessible by Admin roles.
	// security:
	// - Bearer: []
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: limit
	//   in: query
	//   type: integer
	//   required: false
	// - name: page
	//   in: query
	//   type: integer
	//   required: false
	// - name: sort_key
	//   in: query
	//   type: string
	//   required: false
	// - name: sort_val
	//   in: query
	//   type: integer
	//   required: false
	// responses:
	//   "200":
	//     "$ref": "#/responses/listUserResp"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("", h.list)

	// swagger:operation GET /v1/users/{Id} users getUserReq
	// ---
	// summary: Returns a single user.
	// description: |
	//   Returns a single user by its ID.
	//   Only accessible by Admin roles. For non-admin access to User info, use the `GET /me` route.
	// security:
	// - Bearer: []
	// produces:
	//  - application/json
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of user
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "schema":
	//      "$ref": "#/definitions/User"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("/:id", h.view)

	// swagger:operation DELETE /v1/users/{Id} users deleteUserReq
	// ---
	// summary: Deletes a user.
	// description: |
	//   Deletes a user with requested ID.
	//   Only accessible by Admin roles.
	// security:
	// - Bearer: []
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of user
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ok"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.DELETE("/:id", h.delete)

	// swagger:operation PATCH /v1/users/{Id} users updateUserReq
	// ---
	// summary: Updates user's contact information.
	// description: |
	//   Updates user's contact information, including the following fields - first name, last name, email.
	//   Non-verified updates to an email can only be done by an Admin role. To update email as a non-admin user,
	//   retrieve and specify an email verification code via `GET /v1/users/{Id}/email-verification`.
	// security:
	// - Bearer: []
	// consumes:
	//  - application/json
	// produces:
	//  - application/json
	// responses:
	//   "200":
	//     "schema":
	//      "$ref": "#/definitions/User"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.PATCH("/:id", h.update)

	// swagger:operation POST /v1/users/{Id}/profile users updateUserProfileReq
	// ---
	// summary: Updates a user's profile picture.
	// description: Updates a user profile picture with requested ID. Uploaded file should be an image in JPEG or PNG format.
	// security:
	// - Bearer: []
	// consumes:
	//  - multipart/form-data
	// produces:
	//  - application/json
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of user
	//   type: string
	//   required: true
	// - name: file
	//   in: formData
	//   description: Profile picture to upload.
	//   type: file
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ok"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.POST("/:id/profile", h.profile)

	// swagger:operation DELETE /v1/users/{Id}/profile users deleteUserProfileReq
	// ---
	// summary: Deletes a user's profile picture.
	// description: Deletes a user profile picture with requested ID.
	// security:
	// - Bearer: []
	// consumes:
	//  - application/json
	// produces:
	//  - application/json
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of user
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ok"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.DELETE("/:id/profile", h.deleteProfile)

	// swagger:operation GET /v1/users/{Id}/email-verification users getUserEmailVerificationCodeReq
	// ---
	// summary: Get user email verification code
	// description: Get user email verification code via email.
	// security:
	// - Bearer: []
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of user
	//   type: string
	//   required: true
	// - name: email
	//   in: query
	//   description: email to verify
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ok"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("/:id/email-verification", h.getEmailVerificationCode)

	// swagger:operation GET /v1/users/{Id}/email-verification-confirm users confirmUserEmailVerificationCodeReq
	// ---
	// summary: Confirm user email verification code
	// description: Confirm user email verification code.
	// security:
	// - Bearer: []
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of user
	//   type: string
	//   required: true
	// - name: email
	//   in: query
	//   description: email to verify
	//   type: string
	//   required: true
	// - name: code
	//   in: query
	//   description: verification code to confirm
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ok"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("/:id/email-verification-confirm", h.confirmEmailVerificationCode)
}

// Custom errors
var (
	ErrPasswordsNotMaching = echo.NewHTTPError(http.StatusBadRequest, "passwords do not match")
)

// User create request
// swagger:parameters createUserReq
type CreateUserReq struct {
	// in: body
	Body struct {
		//	User first name
		FirstName string `json:"first_name" validate:"omitempty,min=1,max=100"`
		//	User last name
		LastName string `json:"last_name" validate:"omitempty,min=1,max=100"`
		//	User username
		Username string `json:"username" validate:"required,min=3,alphanum"`
		//	User password
		Password string `json:"password" validate:"required,min=8"`
		//	User password confirmed
		PasswordConfirm string `json:"password_confirm" validate:"required,min=8"`
		//	User email
		Email string `json:"email" validate:"required,email"`
	}
}

// create is a method for creating a user.
// create is an admin route.
func (h HTTP) create(c echo.Context) error {
	r := new(CreateUserReq).Body

	if err := c.Bind(&r); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	if r.Password != r.PasswordConfirm {
		return c.JSON(ErrPasswordsNotMaching.Code, ErrPasswordsNotMaching)
	}

	// admin is a reserved username
	if r.Username == AdminRole {
		return c.NoContent(400)
	}

	userModel := models.NewUser(r.Username, r.Password, r.Email, r.FirstName, r.LastName)
	usr, err := h.svc.Create(c, userModel)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, usr)
}

// Users list response
// swagger:response listUserResp
type listUserResp struct {
	// in:body
	Body struct {
		Users []models.User `json:"users"`
		Page  int           `json:"page"`
		Count int64         `json:"count"`
	}
}

// list is a method for listing users.
// list is an admin route.
func (h HTTP) list(c echo.Context) error {
	var req models.PaginationReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	result, count, err := h.svc.List(c, req.Transform())
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	resp := listUserResp{struct {
		Users []models.User "json:\"users\""
		Page  int           "json:\"page\""
		Count int64         "json:\"count\""
	}{result, req.Page, count}}

	return c.JSON(http.StatusOK, resp.Body)
}

// view is a method for viewing a user.
// view is an admin route.
func (h HTTP) view(c echo.Context) error {
	id := c.Param("id")
	user := c.Get("current_user").(models.User)

	// Deny if not an admin and attempting to view another user's info
	if user.Username != AdminRole {
		if id != user.ID.Hex() {
			return echo.ErrUnauthorized
		}
	}

	result, err := h.svc.View(c, id)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, result)
}

// delete is a method for deleting a user.
// delete is an admin route.
func (h HTTP) delete(c echo.Context) error {
	id := c.Param("id")

	if err := h.svc.Delete(c, id); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": fmt.Sprintf("User %s deleted", id)})
}

func (h HTTP) profile(c echo.Context) error {
	userID := c.Param("id")
	user := c.Get("current_user").(models.User)

	// Deny if not an admin and attempting to view another user's info
	if user.Username != AdminRole {
		if userID != user.ID.Hex() {
			return echo.ErrUnauthorized
		}
	}

	maxSize := int64(1024000 * 1000 * .5) // allow only 500MB of file size for now

	err := c.Request().ParseMultipartForm(maxSize)
	if err != nil {
		return c.JSON(415, echo.NewHTTPError(415, err.Error()))
	}

	file, headers, err := c.Request().FormFile("file")
	if err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}
	defer file.Close()

	err = h.svc.Profile(c, userID, headers.Filename, file)
	if err != nil {
		log.Errorf("could not upload file; err=%s", err.Error())
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, headers.Filename)
}

func (h HTTP) deleteProfile(c echo.Context) error {
	userID := c.Param("id")
	user := c.Get("current_user").(models.User)

	// Deny if not an admin and attempting to delete another user's info
	if user.Username != AdminRole {
		if userID != user.ID.Hex() {
			return echo.ErrUnauthorized
		}
	}

	err := h.svc.DeleteProfile(c, userID)
	if err != nil {
		log.Errorf("could not delete profile picture; err=%s", err.Error())
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": "Profile picture deleted"})
}

// User update request
// swagger:parameters updateUserReq
type updateUserReq struct {
	// ID of user
	// in: path
	// type: string
	// required: true
	Id string `param:"Id" validate:"required"`
	// in: body
	Body struct {
		FirstName             string `query:"first_name" json:"first_name,omitempty" validate:"omitempty,min=1,max=100"`
		LastName              string `query:"last_name" json:"last_name,omitempty" validate:"omitempty,min=1,max=100"`
		Email                 string `query:"email" json:"email,omitempty" validate:"omitempty,email"`
		EmailVerificationCode string `query:"email_verification_code" json:"email_verification_code,omitempty" validate:"omitempty,min=2"`
	}
}

// update is a method for updating a users information.
func (h HTTP) update(c echo.Context) error {
	id := c.Param("id")
	user := c.Get("current_user").(models.User)

	req := new(updateUserReq).Body
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	// Deny if not an admin and attempting to modify another user's info
	if user.Username != AdminRole {
		if id != user.ID.Hex() {
			return echo.ErrUnauthorized
		}
	}

	usr, err := h.svc.Update(c, Update{
		ID:                    id,
		FirstName:             req.FirstName,
		LastName:              req.LastName,
		Email:                 req.Email,
		EmailVerificationCode: req.EmailVerificationCode,
		IsAdmin:               user.Username == AdminRole,
	})
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, usr)
}

func (h *HTTP) getEmailVerificationCode(c echo.Context) error {
	userID := c.Param("id")
	user := c.Get("current_user").(models.User)

	// Deny if not an admin and attempting to view another user's info
	if user.Username != AdminRole {
		if userID != user.ID.Hex() {
			return echo.ErrUnauthorized
		}
	}

	email := c.QueryParam("email")
	if email == "" {
		return c.JSON(400, echo.NewHTTPError(400, "`email` is a required parameter"))
	}

	if err := h.svc.EmailVerificationCode(c, userID, email); err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	log.Debugf("successfully mailed email verification code")

	return c.JSON(http.StatusOK, "Please check your mail for email verification code")
}

func (h *HTTP) confirmEmailVerificationCode(c echo.Context) error {
	userID := c.Param("id")
	user := c.Get("current_user").(models.User)

	// Deny if not an admin and attempting to view another user's info
	if user.Username != AdminRole {
		if userID != user.ID.Hex() {
			return echo.ErrUnauthorized
		}
	}

	email := c.QueryParam("email")
	code := c.QueryParam("code")
	if email == "" || code == "" {
		return c.JSON(400, echo.NewHTTPError(400, "`email` and `code` are required parameters"))
	}

	if err := h.svc.EmailVerificationCodeConfirm(c, email, code); err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": fmt.Sprintf("Code for email %s valid", email)})
}
