package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"

	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/mail"
)

// Custom errors
var (
	// ErrUnauthorized (401) is returned when user is not authorized
	ErrUnauthorized     = echo.ErrUnauthorized
	ErrBadRequest       = echo.ErrBadRequest
	ErrInsecurePassword = echo.NewHTTPError(http.StatusBadRequest, "password does not meet minimum required complexity and/or character length (8)")
)

// Token holds authentication token details with refresh token
//
// swagger:model Token
type Token struct {
	// User authentication JWT token
	//
	// min: 1
	AccessToken string `json:"access_token"`
	// User authentication refresh token
	//
	RefreshToken string `json:"refresh_token"`
}

// Authenticate tries to authenticate the user provided by username and password
func (a Auth) Authenticate(c echo.Context, user, pass string) (Token, error) {
	u, err := a.platform.UserDB.FindByUsername(a.db, user)
	if err != nil {
		return Token{}, err
	}

	if !a.sec.HashMatchesPassword(u.Password, pass) {
		return Token{}, ErrUnauthorized
	}

	accessToken, refreshToken, err := a.token.GenerateTokenPair(&u)
	if err != nil {
		return Token{}, ErrUnauthorized
	}

	u.LastLogin = time.Now()
	if err := a.platform.UserDB.UpdateLogin(a.db, u); err != nil {
		return Token{}, err
	}

	return Token{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

// Refresh refreshes jwt token and puts new claims inside
func (a Auth) Refresh(c echo.Context, refreshToken string) (Token, error) {
	claims, err := a.token.ParseToken(refreshToken)
	if err != nil {
		return Token{}, ErrUnauthorized
	}

	user, err := a.token.ValidateToken(claims, true)
	if err != nil {
		return Token{}, ErrUnauthorized
	}

	accessToken, refreshToken, err := a.token.GenerateTokenPair(&user)
	if err != nil {
		return Token{}, ErrUnauthorized
	}

	return Token{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

// Send user password reset code
func (a Auth) SendResetCode(c echo.Context, userEmail string) error {
	u, err := a.platform.UserDB.FindByEmail(a.db, userEmail)
	if err != nil {
		return err
	}

	// Send verification mail
	mailType := mail.PassReset
	mailData := &mail.PortalMailData{
		Username: u.Username,
		Code:     common.GenerateRandomString(8),
	}

	mailReq := a.mail.NewMail(a.mail.PassResetEmail, []string{u.Email}, a.mail.PassResetSubject, mailType, mailData)
	err = a.mail.SendMail(mailReq)
	if err != nil {
		log.Errorf("unable to send mail", "error", err)
		return err
	}

	// cache the password reset code
	key, val := fmt.Sprintf("%s-%d", u.Email, mail.PassReset), mailData.Code
	exp := time.Minute * time.Duration(a.mail.PassResetCodeExpiration)
	if err := a.cache.Set(key, val, exp); err != nil {
		return err
	}

	return nil
}

func (a Auth) ResetPassword(c echo.Context, email, resetToken, newPassword string) error {
	u, err := a.platform.UserDB.FindByEmail(a.db, email)
	if err != nil {
		log.Errorf("user email not found; email=%s", email)
		return ErrBadRequest
	}

	// Check reset token
	key := fmt.Sprintf("%s-%d", u.Email, mail.PassReset)
	token, err := a.cache.Get(key)
	if err != nil {
		log.Errorf("reset token not found; key=%s", key)
		return ErrBadRequest
	}

	if token != strings.TrimSpace(resetToken) {
		log.Errorf("password reset code mismatch; provided=%s expected=%s", resetToken, token)
		return ErrBadRequest
	}

	if !a.sec.Password(newPassword, u.FirstName, u.LastName, u.Username, u.Email) {
		return ErrInsecurePassword
	}

	u.ChangePassword(a.sec.Hash(newPassword))

	return a.platform.UserDB.UpdatePassword(a.db, u)
}

// Me returns info about currently logged user
func (a Auth) Me(c echo.Context, userid string) (models.User, error) {
	return a.platform.UserDB.View(a.db, userid)
}
