package user

import (
	"mime/multipart"

	"github.com/labstack/echo/v4"

	cache "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/cache"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	mail "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/mail"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
	auth "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/middleware/auth"
)

// Service represents user application interface
type Service interface {
	Create(echo.Context, models.User) (models.User, error)
	List(echo.Context, models.Pagination) ([]models.User, int64, error)
	View(echo.Context, string) (models.User, error)
	Delete(echo.Context, string) error
	Update(echo.Context, Update) (models.User, error)
	Profile(echo.Context, string, string, multipart.File) error
	DeleteProfile(echo.Context, string) error
	EmailVerificationCode(echo.Context, string, string) error
	EmailVerificationCodeConfirm(echo.Context, string, string) error
}

// New creates new user application service
func New(db *db.DB, platform *platform.Platform, sec Securer, enforcer *auth.Enforcer, mail *mail.SGPortalMailService, cache *cache.Cache) *User {
	return &User{db: db, platform: platform, sec: sec, enforcer: enforcer, mail: mail, cache: cache}
}

// Initialize initalizes User application service with defaults
func Initialize(db *db.DB, platform *platform.Platform, sec Securer, enforcer *auth.Enforcer, mail *mail.SGPortalMailService, cache *cache.Cache) *User {
	return New(db, platform, sec, enforcer, mail, cache)
}

// User represents user application service
type User struct {
	db       *db.DB
	cache    *cache.Cache
	platform *platform.Platform
	sec      Securer
	enforcer *auth.Enforcer
	mail     *mail.SGPortalMailService
}

// Securer represents security interface
type Securer interface {
	Hash(string) string
}
