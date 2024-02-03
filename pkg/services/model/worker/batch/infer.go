/*
 * File: infer.go
 * Project: batch
 * File Created: Monday, 12th September 2022 5:16:16 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package batch

import (
	"errors"
	"path"

	"github.com/aws/aws-sdk-go-v2/service/sagemakerruntime"

	blob "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/blob"
	image "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/image"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	sage "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/sage"
	realtime "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/sage/realtime"
)

const (
	// Confidence threshold for burning in bounding boxes.
	BoundingBoxConfidenceThreshold = .9
)

var ErrEmptyPredictions = errors.New("no predictions found")

type inferArgs struct {
	content                models.Content
	model                  models.Model
	blob                   *blob.Blob
	sagemakerRuntimeClient *sagemakerruntime.Client
	thumbnailSize          int
}

func (args inferArgs) infer() (models.Prediction, error) {

	contentBytes, _, err := args.blob.Downloader.Download(args.content.StoredDir, args.content.StoredPath)
	if err != nil {
		return models.Prediction{ContentID: args.content.ID}, err
	}

	imageStats, err := image.GetStats(contentBytes)
	if err != nil {
		return models.Prediction{ContentID: args.content.ID}, err
	}

	detectionType, err := models.ProjectAnnotationTypeFromString(args.model.Metadata["type"].(string))
	if err != nil {
		return models.Prediction{ContentID: args.content.ID}, err
	}

	result, err := realtime.New(contentBytes, args.model.Deployment.EndpointName, detectionType.String(), args.sagemakerRuntimeClient)
	if err != nil {
		return models.Prediction{ContentID: args.content.ID}, err
	}
	result.UpdateFilename(path.Base(args.content.Name))

	formattedResult := result.ToFormattedResult(args.model.IntegerMapping, sage.Const_BatchPredictionConfidenceThreshold, imageStats)

	if len(formattedResult.Predictions) > 0 {
		// Add b64 thumbnail
		b64Img := ""
		if detectionType == models.ProjectAnnotationTypeClassification {
			b64Img, _, err = image.Thumbnail(contentBytes, args.thumbnailSize, args.thumbnailSize)
			if err != nil {
				return models.Prediction{ContentID: args.content.ID}, err
			}
		} else if detectionType == models.ProjectAnnotationTypeBoundingBox {
			boundingBoxes := []image.BoundingBox{}
			for _, box := range formattedResult.Predictions {
				if box.Confidence >= BoundingBoxConfidenceThreshold {
					boundingBoxes = append(boundingBoxes, image.BoundingBox{
						Xmin:      box.BoundingBox["xmin"].(int),
						Xmax:      box.BoundingBox["xmax"].(int),
						Ymin:      box.BoundingBox["ymin"].(int),
						Ymax:      box.BoundingBox["ymax"].(int),
						ClassName: box.ClassName,
					})
				}
			}
			b64Img, _, err = image.ThumbnailBoundingBox(
				contentBytes,
				args.thumbnailSize,
				args.thumbnailSize,
				boundingBoxes,
			)
			if err != nil {
				return models.Prediction{ContentID: args.content.ID}, err
			}
		}
		return models.NewPrediction(
			args.model.UserID,
			args.model.ID.Hex(),
			args.content.ID,
			detectionType,
			b64Img,
			formattedResult.Predictions,
		), nil
	}
	return models.Prediction{ContentID: args.content.ID}, ErrEmptyPredictions

}
