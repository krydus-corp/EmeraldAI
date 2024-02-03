/*
 * File: counts.go
 * Project: common
 * File Created: Tuesday, 16th August 2022 9:57:27 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package train

import (
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

// SplitCounts is a struct for modeling train/validation/test counts
type SplitCounts struct {
	TrainCount      int
	ValidationCount int
	TestCount       int
}

// GetSplitCounts is a helper method for determining train/validation/test based on dataset split
func GetSplitCounts(dataset *models.Dataset, trainSplit, validationSplit, testSplit float64, annotationDB platform.Annotation, db *db.DB) (SplitCounts, error) {
	annotationCount, err := annotationDB.CountAnnotations(db, dataset.UserID, dataset.ID.Hex())
	if err != nil {
		return SplitCounts{}, err
	}

	trainCount := int(trainSplit * float64(*annotationCount))
	validationCount := int(validationSplit * float64(*annotationCount))
	testCount := 0
	remaining := int(*annotationCount) - int(trainCount) - int(validationCount)
	if testSplit == 0 {
		trainCount += remaining
	} else {
		testCount = remaining
	}

	return SplitCounts{
		trainCount,
		validationCount,
		testCount,
	}, nil
}
