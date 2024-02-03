package annotation

import (
	"github.com/labstack/echo/v4"

	blob "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/blob"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

// Service represents annotation application interface
type Service interface {
	Create(echo.Context, models.Annotation, int) (*models.Annotation, error)
	List(echo.Context, string, string, string, models.Pagination) ([]models.Annotation, int64, error)
	View(echo.Context, string, string) (*models.Annotation, error)
	Query(echo.Context, string, models.Query) ([]models.Annotation, int64, error)
	Delete(echo.Context, string, ...string) error
	Statistics(echo.Context, string, string, string, *string) (*Statistics, error)
}

// New creates new annotation application service
func New(db *db.DB, platform *platform.Platform, blob *blob.Blob) *Annotation {
	return &Annotation{db: db, platform: platform, blob: blob}
}

// Initialize initalizes Annotation application service with defaults
func Initialize(db *db.DB, platform *platform.Platform, blob *blob.Blob) *Annotation {
	return New(db, platform, blob)
}

// Annotation represents annotation application service
type Annotation struct {
	db       *db.DB
	platform *platform.Platform
	blob     *blob.Blob
}

// Statistics models all annotation statistics
// swagger:model AnnotationStatistics
type Statistics struct {
	TotalAnnotatedImages       *int64                          `json:"total_annotated_images,omitempty"`        // Total content with 1 or more tags associated
	TotalUnannotatedImages     *int64                          `json:"total_unannotated_images,omitempty"`      // Total content with 1 or more tags associated
	TotalAnnotations           *int64                          `json:"total_annotations,omitempty"`             // Total annotations across all images
	AverageAnnotationsPerImage *float64                        `json:"average_annotations_per_image,omitempty"` // Average number of tags per each image
	AnnotatedNullTags          *int64                          `json:"annotated_null_images,omitempty"`         // Annotated images with no tag associations
	AnnotationsPerClass        *map[string]annotationClassData `json:"annotations_per_class,omitempty"`         // Total content associated with each class -> map[tagid]annotationClassData
	AverageImageHeightPixels   *int64                          `json:"average_image_height_pixels,omitempty"`   // Average image height
	AverageImageWidthPixels    *int64                          `json:"average_image_width_pixels,omitempty"`    // Average image width
	MinImageHeightPixels       *int64                          `json:"min_image_height_pixels,omitempty"`       // Min image height
	MinImageWidthPixels        *int64                          `json:"min_image_width_pixels,omitempty"`        // Min image width
	MaxImageHeightPixels       *int64                          `json:"max_image_height_pixels,omitempty"`       // Max image height
	MaxImageWidthPixels        *int64                          `json:"max_image_width_pixels,omitempty"`        // Max image width
	AverageImageSizeBytes      *int64                          `json:"average_image_size_bytes,omitempty"`      // Average image size in bytes
	MinImageSizeBytes          *int64                          `json:"min_image_size_bytes,omitempty"`          // Minimum image size in bytes
	MaxImageSizeBytes          *int64                          `json:"max_image_size_bytes,omitempty"`          // Maximum image size in bytes
	DimensionInsights          *annotationDimensionInsights    `json:"insights,omitempty"`                      // Annotation Insights
}

type annotationClassData struct {
	Name    string  `json:"name"`
	Count   int     `json:"count"`
	Balance string  `json:"balance"`
	Zscore  float64 `json:"zscore"`
}

type annotationDimensionInsights struct {
	Dimensions               []annotationDimensions             `json:"dimensions"`
	SizeDistributions        annotationSizeDistributions        `json:"size_distributions"`
	AspectRatioDistributions annotationAspectRatioDistributions `json:"aspect_ratio_distributions"`
	MedianHeight             float64                            `json:"median_height"`
	MedianWidth              float64                            `json:"median_width"`
}

type annotationDimensions struct {
	Height int `json:"height"`
	Width  int `json:"width"`
}

type annotationSizeDistributions struct {
	Tiny   annotationDistributionData `json:"tiny"`   // < 100 x 100
	Small  annotationDistributionData `json:"small"`  // >= 100 x 100
	Medium annotationDistributionData `json:"medium"` // >= 256 x 256
	Large  annotationDistributionData `json:"large"`  // >= 720 x 720
	Jumbo  annotationDistributionData `json:"jumbo"`  // >= 1024 x 1024
}

type annotationAspectRatioDistributions struct {
	Tall   annotationDistributionData `json:"tall"`
	Wide   annotationDistributionData `json:"wide"`
	Square annotationDistributionData `json:"square"`
}
type annotationDistributionData struct {
	Count   int    `json:"count"`
	Balance string `json:"balance"`
}
