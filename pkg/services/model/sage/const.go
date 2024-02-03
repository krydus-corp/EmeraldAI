/*
 * File: const.go
 * Project: sage
 * File Created: Sunday, 28th August 2022 4:23:20 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package sage

import (
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
)

const (
	Const_ClassificationObjectiveMetric                = "validation:accuracy"
	Const_ObjectDetectionObjectiveMetric               = "validation:mAP"
	Const_BatchPredictionConfidenceThreshold           = 0.10
	Const_DefaultRealtimePredictionConfidenceThreshold = 0.85
)

func ObjectiveMetricName(algorithm models.ProjectAnnotationType) string {
	switch algorithm {
	case models.ProjectAnnotationTypeClassification:
		return Const_ClassificationObjectiveMetric
	case models.ProjectAnnotationTypeBoundingBox:
		return Const_ObjectDetectionObjectiveMetric
	default:
		return ""
	}
}
