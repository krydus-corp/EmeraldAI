package auth

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"

	errs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/error"
)

// Custom errors
var (
	ErrPasswordsNotMaching = echo.NewHTTPError(http.StatusBadRequest, "passwords do not match")
)

// HTTP represents auth http service
type HTTP struct {
	svc Service
}

// NewHTTP creates new auth http service
func NewHTTP(svc Service, e *echo.Echo, mw ...echo.MiddlewareFunc) {
	h := HTTP{svc}
	// swagger:operation POST /login auth loginReq
	// ---
	// summary: Logs in user.
	// description: Logs in user by username and password.
	// consumes:
	//  - application/json
	// produces:
	//  - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/tokenResp"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	e.POST("/login", h.login)
	// swagger:operation GET /refresh/{token} auth refreshReq
	// ---
	// summary: Refreshes jwt token.
	// description: Refreshes jwt token by checking at database whether refresh token exists. Returns new access and refresh tokens.
	// produces:
	//  - application/json
	// parameters:
	// - name: token
	//   in: path
	//   description: refresh token
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/tokenResp"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	e.GET("/refresh/:token", h.refresh)

	// swagger:operation GET /password-reset auth passwordResetCodeReq
	// ---
	// summary: Get user password reset code
	// description: Get user reset password code via email.
	// parameters:
	// - name: email
	//   in: query
	//   description: email of user
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
	e.GET("/password-reset", h.resetCode)

	// swagger:operation POST /password-reset auth passwordResetReq
	// ---
	// summary: Reset user password.
	// description: |
	//   Reset user password using reset code sent via email.
	//   Passwords strength is analyzed utilizing [zxcvbn](https://github.com/dropbox/zxcvbn) library.
	//   It will reject common passwords, common names and surnames according to US census data, popular English words from Wikipedia and US television and movies, and other common patterns like dates, repeats (aaa), sequences (abcd), keyboard patterns (qwertyuiop), and l33t speak.
	//   Password have a minimum required length of 8 characters.
	// consumes:
	//  - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/ok"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	e.POST("/password-reset", h.resetPassword)

	// swagger:operation GET /me auth meReq
	// ---
	// summary: Gets user's info.
	// description: Gets user's info from session.
	// security:
	// - Bearer: []
	// produces:
	//  - application/json
	// responses:
	//   "200":
	//     "schema":
	//      "$ref": "#/definitions/User"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	e.GET("/me", h.me, mw...)
}

// Token response
// swagger:response tokenResp
//
//lint:ignore U1000 ignore, used for swagger spec
type tokenResp struct {
	// in:body
	Body struct {
		*Token
	}
}

// User login request
// swagger:parameters loginReq
type loginReq struct {
	// in:body
	Body struct {
		//	Login username
		//
		Username string `json:"username" validate:"required"`
		//	Login password
		//
		Password string `json:"password" validate:"required"`
	}
}

func (h *HTTP) login(c echo.Context) error {
	cred := new(loginReq).Body
	if err := c.Bind(&cred); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}
	token, err := h.svc.Authenticate(c, cred.Username, cred.Password)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}
	return c.JSON(http.StatusOK, token)
}

func (h *HTTP) refresh(c echo.Context) error {
	token, err := h.svc.Refresh(c, c.Param("token"))
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}
	return c.JSON(http.StatusOK, token)
}

func (h *HTTP) me(c echo.Context) error {
	user := c.Get("current_user").(models.User)

	user, err := h.svc.Me(c, user.ID.Hex())
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}
	return c.JSON(http.StatusOK, user)
}

func (h *HTTP) resetCode(c echo.Context) error {

	email := c.QueryParam("email")
	if email == "" {
		return c.JSON(400, echo.NewHTTPError(400, "`email` is a required parameter"))
	}

	if err := h.svc.SendResetCode(c, email); err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": "Please check your mail for password reset code"})
}

// swagger:parameters passwordResetReq
//
//lint:ignore U1000 ignore, used for swagger spec
type passwordResetReq struct {
	// in:body
	Body struct {
		//	User email
		Email string `query:"email" json:"email" validate:"required"`
		//	User reset token
		ResetToken string `query:"token" json:"token" validate:"required"`
		// Old password
		NewPassword string `query:"new_password" json:"new_password" validate:"required,min=8"`
		// New password confirmed
		NewPasswordConfirm string `query:"new_password_confirm" json:"new_password_confirm" validate:"required"`
	}
}

func (h *HTTP) resetPassword(c echo.Context) error {
	p := new(passwordResetReq).Body

	if err := c.Bind(&p); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	if p.NewPassword != p.NewPasswordConfirm {
		return ErrPasswordsNotMaching
	}

	if err := h.svc.ResetPassword(c, p.Email, p.ResetToken, p.NewPassword); err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": "Password updated"})
}
