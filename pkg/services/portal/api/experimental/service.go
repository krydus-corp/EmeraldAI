package experimental

import (
	"github.com/labstack/echo/v4"

	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

// Service represents user application interface
type Service interface {
	Version(echo.Context) string
}

// New creates new user application service
func New(db *db.DB, platform *platform.Platform) *Experimental {
	return &Experimental{db: db, platform: platform}
}

// Initialize initializes the Experimental application service with defaults
func Initialize(db *db.DB, platform *platform.Platform) *Experimental {
	return New(db, platform)
}

// Experimental represents experimental application service
type Experimental struct {
	db       *db.DB
	platform *platform.Platform
}
