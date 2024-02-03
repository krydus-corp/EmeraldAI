package password

import (
	"net/http"

	"github.com/labstack/echo/v4"

	errs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/error"
)

// HTTP represents password http transport service
type HTTP struct {
	svc Service
}

// NewHTTP creates new password http service
func NewHTTP(svc Service, er *echo.Group) {
	h := HTTP{svc}
	pr := er.Group("/password")

	// swagger:operation PATCH /v1/password/{Id} password changePasswordReq
	// ---
	// summary: Changes user's password.
	// description: |
	//  If user's old password is correct, it will be replaced with new.
	//  Passwords strength is analyzed utilizing [zxcvbn](https://github.com/dropbox/zxcvbn) library.
	//  It will reject common passwords, common names and surnames according to US census data, popular English words from Wikipedia and US television and movies, and other common patterns like dates, repeats (aaa), sequences (abcd), keyboard patterns (qwertyuiop), and l33t speak.
	//  Password have a minimum required length of 8 characters.
	// security:
	// - Bearer: []
	// responses:
	//   "200":
	//     "$ref": "#/responses/ok"
	//   "500":
	//     "$ref": "#/responses/err"
	pr.PATCH("/:id", h.change)
}

// Custom errors
var (
	ErrPasswordsNotMaching = echo.NewHTTPError(http.StatusBadRequest, "passwords do not match")
)

// Password change request
// swagger:parameters changePasswordReq
type changePasswordReq struct {
	// ID of user
	// in: path
	// type: string
	// required: true
	Id string `param:"Id" validate:"required"`
	// in: body
	Body struct {
		// ID of user
		ID int `json:"-"`
		// Old password
		OldPassword string `json:"old_password" validate:"required"`
		// New password
		NewPassword string `json:"new_password" validate:"required,min=8"`
		// New password confirmed
		NewPasswordConfirm string `json:"new_password_confirm" validate:"required,min=8"`
	}
}

func (h *HTTP) change(c echo.Context) error {
	id := c.Param("id")
	p := new(changePasswordReq).Body

	if err := c.Bind(&p); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	if p.NewPassword != p.NewPasswordConfirm {
		return c.JSON(ErrPasswordsNotMaching.Code, ErrPasswordsNotMaching)
	}

	if err := h.svc.Change(c, id, p.OldPassword, p.NewPassword); err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": "Password updated"})
}
