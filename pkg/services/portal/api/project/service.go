package project

import (
	"mime/multipart"

	"github.com/labstack/echo/v4"

	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/blob"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/api/project/upload"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/config"

	awssqs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/aws/sqs"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
)

// New creates new project application service
func New(db *db.DB, platform *platform.Platform, blob *blob.Blob, garbagePublisher awssqs.Publisher) *Project {
	return &Project{
		db:       db,
		platform: platform,
		blob:     blob,

		garbagePublisher: garbagePublisher,
	}
}

// Initialize initializes Project application service with defaults
func Initialize(db *db.DB, platform *platform.Platform, blob *blob.Blob, cfg config.Application) (*Project, error) {
	garbagePublisher, err := awssqs.NewPublisher(&awssqs.Config{}, cfg.GarbageJobQueueName)
	if err != nil {
		return nil, err
	}
	return New(db, platform, blob, garbagePublisher), nil
}

// Service represents project application interface
type Service interface {
	Create(echo.Context, *models.Project, *models.Dataset) (models.Project, error)
	List(echo.Context, string, models.Pagination) ([]models.Project, int64, error)
	View(echo.Context, string, string) (models.Project, error)
	Query(echo.Context, string, models.Query) ([]models.Project, int64, error)
	Delete(echo.Context, string, string) error
	Profile(echo.Context, string, string, string, *multipart.File) (string, error)
	Update(echo.Context, Update) (models.Project, error)
	Upload(echo.Context, CreateUploadReq) (*upload.Report, error)
}

// Project represents project application service
type Project struct {
	db       *db.DB
	blob     *blob.Blob
	platform *platform.Platform

	garbagePublisher awssqs.Publisher
}
