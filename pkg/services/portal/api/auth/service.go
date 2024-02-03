package auth

import (
	"github.com/labstack/echo/v4"

	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/cache"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"

	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/mail"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/jwt"
)

// New creates new iam service
func New(db *db.DB, cache *cache.Cache, platform *platform.Platform, token *jwt.Service, mail *mail.SGPortalMailService, sec Securer) Auth {
	return Auth{
		db:       db,
		platform: platform,
		token:    token,
		sec:      sec,
		cache:    cache,
		mail:     mail,
	}
}

// Initialize initializes auth application service
func Initialize(db *db.DB, cache *cache.Cache, platform *platform.Platform, token *jwt.Service, mail *mail.SGPortalMailService, sec Securer) Auth {
	return New(db, cache, platform, token, mail, sec)
}

// Service represents auth service interface
type Service interface {
	Authenticate(echo.Context, string, string) (Token, error)
	Refresh(echo.Context, string) (Token, error)
	Me(echo.Context, string) (models.User, error)
	SendResetCode(echo.Context, string) error
	ResetPassword(echo.Context, string, string, string) error
}

// Auth represents auth application service
type Auth struct {
	db       *db.DB
	cache    *cache.Cache
	platform *platform.Platform
	token    *jwt.Service
	mail     *mail.SGPortalMailService
	sec      Securer
}

// Securer represents security interface
type Securer interface {
	Hash(string) string
	HashMatchesPassword(string, string) bool
	Password(string, ...string) bool
}
