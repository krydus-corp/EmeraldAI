/*
 * File: service.go
 * Project: export
 * File Created: Saturday, 24th July 2021 3:29:23 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package export

import (
	"github.com/labstack/echo/v4"

	blob "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/blob"
	awssqs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/aws/sqs"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

// New creates new export application service
func New(db *db.DB, platform *platform.Platform, publisher awssqs.Publisher, blob *blob.Blob) *Export {
	return &Export{db: db, platform: platform, publisher: publisher, blob: blob}
}

// Initialize initializes Export application service with defaults
func Initialize(db *db.DB, platform *platform.Platform, blob *blob.Blob, exportJobQueue string) (*Export, error) {
	pub, err := awssqs.NewPublisher(&awssqs.Config{}, exportJobQueue)
	if err != nil {
		return nil, err
	}

	return New(db, platform, pub, blob), nil
}

// Service represents Export application interface
type Service interface {
	Create(echo.Context, string, createExportReq) (models.Export, error)
	View(echo.Context, string, string) (models.Export, error)
	List(echo.Context, string, models.Pagination) ([]models.Export, int64, error)
	Query(echo.Context, string, models.Query) ([]models.Export, int64, error)
	Delete(echo.Context, string, string) error
	GetMetadata(echo.Context, string, string) ([]byte, error)
	GetContent(echo.Context, string, string, string, int) (string, error)
}

// Export represents export application service
type Export struct {
	db        *db.DB
	platform  *platform.Platform
	publisher awssqs.Publisher
	blob      *blob.Blob
}
