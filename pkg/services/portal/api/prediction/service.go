package prediction

import (
	"bytes"

	"github.com/labstack/echo/v4"

	blob "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/blob"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

// Service represents prediction application interface
type Service interface {
	Query(echo.Context, string, models.Query) ([]models.Prediction, int64, error)
	Statistics(echo.Context, string, string, float64, ...string) (*Statistics, error)
	Predictions(echo.Context, string, string, float64, int, int, ...string) ([]models.Prediction, error)
	Heatmap(echo.Context, string, string) (*bytes.Buffer, error)
}

// New creates new prediction application service
func New(db *db.DB, platform *platform.Platform, blob *blob.Blob) *Prediction {
	return &Prediction{db: db, platform: platform, blob: blob}
}

// Initialize initalizes Prediction application service with defaults
func Initialize(db *db.DB, platform *platform.Platform, blob *blob.Blob) *Prediction {
	return New(db, platform, blob)
}

// Prediction represents prediction application service
type Prediction struct {
	db       *db.DB
	platform *platform.Platform
	blob     *blob.Blob
}

// Statistics models prediction statistics, including total and class counts
// swagger:model PredictionStatistics
type Statistics struct {
	TotalPredictions    int64                    `json:"total_predictions"`     // Total predictions
	PredictionsPerClass []map[string]interface{} `json:"predictions_per_class"` // Prediction counts for each class -> map[tagid]predictionClassData
}
