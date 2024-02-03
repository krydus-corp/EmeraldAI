// Package prediction contains prediction application services
package prediction

import (
	"bytes"
	"image"
	"image/jpeg"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/sync/errgroup"

	common "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	emldimage "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/image"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
)

const (
	DefaultUncertaintySampleCount int     = 10
	DefaultUncertaintyThreshold   float64 = .9
)

func (p Prediction) Query(ctx echo.Context, userid string, q models.Query) ([]models.Prediction, int64, error) {
	return p.platform.PredictionDB.Query(p.db, userid, q)
}

// Statistics returns prediction stats, including total predictions and predictions per class
func (p Prediction) Statistics(c echo.Context, userid, modelid string, threshold float64, tagIDs ...string) (*Statistics, error) {
	// Check model exists
	model, err := p.platform.ModelDB.View(p.db, userid, modelid)
	if err != nil {
		return nil, err
	}

	stats := Statistics{
		TotalPredictions:    0,
		PredictionsPerClass: make([]map[string]interface{}, 0),
	}

	// Default threshold
	if threshold == 0 {
		threshold = DefaultUncertaintyThreshold
	}

	// Parallelize stat requests
	// Note: Synchronization not required as, while we are performing write access,
	// they are all on different fields within `stats` and different struct fields have different memory locations.
	var g errgroup.Group

	// Total predictions
	g.Go(func() error {
		var err error
		if stats.TotalPredictions, err = p.platform.PredictionDB.Count(p.db, model); err != nil {
			return err
		}
		return nil
	})
	// Predictions per class
	g.Go(func() error {
		var err error
		if stats.PredictionsPerClass, err = p.platform.PredictionDB.PredictionsPerClass(p.db, model, threshold, tagIDs...); err != nil {
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &stats, nil
}

func (p Prediction) Predictions(c echo.Context, userid, modelid string, threshold float64, sampleCount, thumbnailSize int, tagIDs ...string) ([]models.Prediction, error) {
	var results []models.Prediction

	// Check model exists
	model, err := p.platform.ModelDB.View(p.db, userid, modelid)
	if err != nil {
		return nil, err
	}

	// Check model project for type
	project, err := p.platform.ProjectDB.View(p.db, userid, model.ProjectID)
	if err != nil {
		return nil, err
	}

	// Get list of all tags
	options := options.FindOptions{AllowDiskUse: common.Ptr(true)}
	tagList, err := p.platform.TagDB.ListAll(p.db, userid, model.DatasetID, &options)
	if err != nil {
		return nil, err
	}

	// Sample all tagids
	if len(tagIDs) == 0 {
		tagNames := []string{}
		for _, tag := range tagList {
			tagNames = append(tagNames, tag.Name)
		}

		predictions, err := p.platform.PredictionDB.Sample(p.db, userid, modelid, tagNames, threshold, sampleCount)
		if err != nil {
			return nil, err
		}
		results = append(results, predictions...)
	} else {
		tagNames := []string{}
		for _, tagID := range tagIDs {
			tag, err := p.platform.TagDB.View(p.db, userid, tagID)
			if err != nil {
				return nil, err
			}
			tagNames = append(tagNames, tag.Name)
		}

		predictions, err := p.platform.PredictionDB.Sample(p.db, userid, modelid, tagNames, threshold, sampleCount)
		if err != nil {
			return nil, err
		}

		results = append(results, predictions...)
	}

	// Tag id lookup hash
	tagLookup := make(map[string]string)
	for _, tag := range tagList {
		tagLookup[tag.Name] = tag.ID.Hex()
	}

	// Check if thumbnail size is different than default.
	if thumbnailSize == 0 {
		thumbnailSize = emldimage.BoundingBoxDefaultThumbnailSize
	}

	for i := 0; i < len(results); i++ {
		// Remove predictions < threshold
		// Update tag name
		predictions := []models.PredictionMetadata{}
		boundingBoxes := []emldimage.BoundingBox{}
		for j := 0; j < len(results[i].Predictions); j++ {
			if results[i].Predictions[j].Confidence >= threshold {
				// Only if project type is bounding box.
				if project.AnnotationType == models.ProjectAnnotationTypeBoundingBox.String() {
					boundingBoxes = append(boundingBoxes, emldimage.BoundingBox{
						Xmin:      int(results[i].Predictions[j].BoundingBox["xmin"].(int32)),
						Xmax:      int(results[i].Predictions[j].BoundingBox["xmax"].(int32)),
						Ymin:      int(results[i].Predictions[j].BoundingBox["ymin"].(int32)),
						Ymax:      int(results[i].Predictions[j].BoundingBox["ymax"].(int32)),
						ClassName: results[i].Predictions[j].ClassName,
					})
				}
				results[i].Predictions[j].TagID = tagLookup[results[i].Predictions[j].ClassName]
				predictions = append(predictions, results[i].Predictions[j])
			}
		}

		// Only need to update b64 if not default threshold or size. The default img is already generated.
		if threshold != DefaultUncertaintyThreshold || thumbnailSize != emldimage.BoundingBoxDefaultThumbnailSize || results[i].Base64Image == "" {
			content, err := p.platform.ContentDB.View(p.db, userid, results[i].ContentID)
			if err != nil {
				return nil, err
			}
			contentBytes, err := p.blob.Get(content.StoredDir, content.StoredPath)
			if err != nil {
				return nil, err
			}

			// Get new b64 thumbnail images.
			b64Img := ""
			if project.AnnotationType == models.ProjectAnnotationTypeBoundingBox.String() {
				b64Img, _, err = emldimage.ThumbnailBoundingBox(
					contentBytes,
					thumbnailSize,
					thumbnailSize,
					boundingBoxes,
				)
				if err != nil {
					return nil, err
				}
			} else {
				b64Img, _, err = emldimage.Thumbnail(
					contentBytes,
					thumbnailSize,
					thumbnailSize,
				)
				if err != nil {
					return nil, err
				}
			}
			results[i].Base64Image = b64Img
		}
		results[i].Predictions = predictions
	}

	return results, nil
}

func (p Prediction) Heatmap(ctx echo.Context, userid, predictionid string) (*bytes.Buffer, error) {
	// Get prediction
	prediction, err := p.platform.PredictionDB.View(p.db, userid, predictionid)
	if err != nil {
		return nil, err
	}

	// Get content
	content, err := p.platform.ContentDB.View(p.db, userid, prediction.ContentID)
	if err != nil {
		return nil, err
	}

	points := []emldimage.DataPoint{}
	for _, prediction := range prediction.Predictions {
		xmin := prediction.BoundingBox["xmin"].(int32)
		ymin := prediction.BoundingBox["ymin"].(int32)
		xmax := prediction.BoundingBox["xmax"].(int32)
		ymax := prediction.BoundingBox["ymax"].(int32)

		points = append(
			points, emldimage.P(
				float64(xmin),
				float64(ymin),
				int(xmax-xmin),
				int(ymax-ymin)),
		)
	}

	contentBytes, err := p.blob.Get(content.StoredDir, content.StoredPath)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(contentBytes))
	if err != nil {
		return nil, err
	}

	heatmap := emldimage.Heatmap(img.Bounds(), points, nil, 100, emldimage.Classic, emldimage.OverlayShapeDot)

	overlay := emldimage.AddOverlay(heatmap, img, 0.5)

	buf := new(bytes.Buffer)

	opts := jpeg.Options{Quality: 100}
	err = jpeg.Encode(buf, overlay, &opts)
	if err != nil {
		return nil, err
	}

	return buf, nil
}
