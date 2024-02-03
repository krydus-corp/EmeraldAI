package dataset

import (
	"github.com/labstack/echo/v4"

	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

// New creates new dataset application service
func New(db *db.DB, platform *platform.Platform) *Dataset {
	return &Dataset{db: db, platform: platform}
}

// Initialize initializes Dataset application service with defaults
func Initialize(db *db.DB, platform *platform.Platform) (*Dataset, error) {
	return New(db, platform), nil
}

// Service represents Dataset application interface
type Service interface {
	View(echo.Context, string, string) (*models.Dataset, error)
	Update(echo.Context, models.Dataset) (*models.Dataset, error)
}

// Dataset represents dataset application service
type Dataset struct {
	db       *db.DB
	platform *platform.Platform
}
