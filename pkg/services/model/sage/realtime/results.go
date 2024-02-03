/*
 * File: results.go
 * Project: realtime
 * File Created: Friday, 26th August 2022 9:44:46 am
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package realtime

import (
	common "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	stats "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/image"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
)

type ModelReturn interface {
	ToFormattedResult(classMap map[string]int, confidenceThreshold float64, stats *stats.Stats) DetectionReturn
	UpdateFilename(string)
}

// Holds the return from a classification model
type ClassificationResult struct {
	Filename string    `json:"-,omitempty"`
	Return   []float64 `json:"classification_predictions,omitempty"`
}

// Holds the return from an object detection model
type ObjectDetectionResult struct {
	Filename string                 `json:"-,omitempty"`
	Return   map[string][][]float64 `json:"object_detection_predictions,omitempty"`
}

// Our formatted detection return
type DetectionReturn struct {
	Filename    string                      `json:"filename"`
	Predictions []models.PredictionMetadata `json:"predictions"`
}

func (c *ClassificationResult) ToFormattedResult(classMap map[string]int, confidenceThreshold float64, stats *stats.Stats) DetectionReturn {
	classMapReversed := common.ReverseMapStringInt(classMap)
	fmtResult := DetectionReturn{Filename: c.Filename, Predictions: []models.PredictionMetadata{}}

	if len(c.Return) > 0 {
		for idx, prediction := range c.Return {
			if prediction < confidenceThreshold {
				continue
			}
			mappedPrediction, ok := classMapReversed[idx]
			if !ok {
				log.Errorf("error mapping prediction to class")
				continue
			}
			fmtResult.Predictions = append(fmtResult.Predictions, models.PredictionMetadata{
				ClassIndex: idx,
				ClassName:  mappedPrediction,
				Confidence: prediction,
			})
		}
	}

	return fmtResult
}

func (o *ObjectDetectionResult) ToFormattedResult(classMap map[string]int, confidenceThreshold float64, stats *stats.Stats) DetectionReturn {
	classMapReversed := common.ReverseMapStringInt(classMap)
	fmtResult := DetectionReturn{Filename: o.Filename, Predictions: []models.PredictionMetadata{}}

	if len(o.Return["prediction"]) > 0 {
		for idx, detection := range o.Return["prediction"] {
			if detection[1] < confidenceThreshold {
				continue
			}
			mappedPrediction, ok := classMapReversed[int(detection[0])]
			if !ok {
				log.Errorf("error mapping prediction to class")
				continue
			}
			fmtResult.Predictions = append(fmtResult.Predictions, models.PredictionMetadata{
				ClassIndex: idx,
				ClassName:  mappedPrediction,
				Confidence: detection[1],
				BoundingBox: map[string]interface{}{
					"xmin": int(detection[2] * float64(stats.Width)),
					"ymin": int(detection[3] * float64(stats.Height)),
					"xmax": int(detection[4] * float64(stats.Width)),
					"ymax": int(detection[5] * float64(stats.Height)),
				},
			})
		}
	}
	return fmtResult
}

func (c *ClassificationResult) UpdateFilename(filename string) {
	c.Filename = filename
}

func (o *ObjectDetectionResult) UpdateFilename(filename string) {
	o.Filename = filename
}
