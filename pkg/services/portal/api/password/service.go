package password

import (
	"github.com/labstack/echo/v4"

	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"

	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

// Service represents password application interface
type Service interface {
	Change(echo.Context, string, string, string) error
}

// New creates new password application service
func New(db *db.DB, platform *platform.Platform, sec Securer) Password {
	return Password{
		db:       db,
		platform: platform,
		sec:      sec,
	}
}

// Initialize initalizes password application service with defaults
func Initialize(db *db.DB, platform *platform.Platform, sec Securer) Password {
	return New(db, platform, sec)
}

// Password represents password application service
type Password struct {
	db       *db.DB
	platform *platform.Platform
	sec      Securer
}

// Securer represents security interface
type Securer interface {
	Hash(string) string
	HashMatchesPassword(string, string) bool
	Password(string, ...string) bool
}
