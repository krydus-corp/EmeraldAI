/*
 * File: service.go
 * Project: content
 * File Created: Sunday, 9th May 2021 12:43:54 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package content

import (
	"github.com/labstack/echo/v4"

	blob "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/blob"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

// New creates new content application service
func New(db *db.DB, platform *platform.Platform, blob *blob.Blob) *Content {
	return &Content{db: db, platform: platform, blob: blob}
}

// Initialize initializes User application service with defaults
func Initialize(db *db.DB, platform *platform.Platform, blob *blob.Blob) (*Content, error) {
	return New(db, platform, blob), nil
}

// Service represents user application interface
type Service interface {
	ListAnnotated(echo.Context, models.Pagination, string, string, string, string, ...string) ([]models.Content, int, error)
	Get(echo.Context, string, string, string, *string, bool) (*models.Content, []byte, error)
	Sample(echo.Context, string, string, int, bool) ([]models.Content, error)
	Query(echo.Context, string, models.Query) ([]models.Content, int64, error)
	Delete(echo.Context, string, string, []string) error
}

// Content represents content application service
type Content struct {
	db       *db.DB
	blob     *blob.Blob
	platform *platform.Platform
}
