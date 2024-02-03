package password

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Custom errors
var (
	ErrIncorrectPassword = echo.NewHTTPError(http.StatusBadRequest, "incorrect old password")
	ErrInsecurePassword  = echo.NewHTTPError(http.StatusBadRequest, "password does not meet minimum required complexity and/or character length (8)")
)

// Change changes user's password
func (p Password) Change(c echo.Context, userID, oldPass, newPass string) error {
	u, err := p.platform.UserDB.View(p.db, userID)
	if err != nil {
		return err
	}

	if !p.sec.HashMatchesPassword(u.Password, oldPass) {
		return ErrIncorrectPassword
	}

	if !p.sec.Password(newPass, u.FirstName, u.LastName, u.Username, u.Email) {
		return ErrInsecurePassword
	}

	u.ChangePassword(p.sec.Hash(newPass))

	return p.platform.UserDB.UpdatePassword(p.db, u)
}
