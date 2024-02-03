package tag

import (
	"github.com/labstack/echo/v4"

	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

// New creates new tag application service
func New(db *db.DB, platform *platform.Platform) *Tag {
	return &Tag{db: db, platform: platform}
}

// Initialize initializes Tag application service with defaults
func Initialize(db *db.DB, platform *platform.Platform) (*Tag, error) {
	return New(db, platform), nil
}

// Service represents Tag application interface
type Service interface {
	Create(echo.Context, models.Tag) (models.Tag, error)
	List(echo.Context, string, string, models.Pagination) ([]models.Tag, int64, error)
	View(echo.Context, string, string) (models.Tag, error)
	Update(echo.Context, Update) (models.Tag, error)
	Query(echo.Context, string, models.Query) ([]models.Tag, int64, error)
	Properties(echo.Context, string, string) ([]string, error)
	Delete(echo.Context, string, string) error
}

// Tag represents tag application service
type Tag struct {
	db       *db.DB
	platform *platform.Platform
}
