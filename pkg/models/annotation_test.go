/*
 * File: annotation_test.go
 * Project: models
 * File Created: Sunday, 25th December 2022 1:41:37 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package models

import (
	"testing"
)

// Test conversion from Pascal VOC to CoCo
// https://albumentations.ai/docs/getting_started/bounding_boxes_augmentation
func TestAnnotationConversion(t *testing.T) {
	metatdata := AnnotationMetadata{
		[]AnnotationDataBoundingBox{
			{
				TagID: "",
				Xmin:  98,
				Ymin:  345,
				Xmax:  420,
				Ymax:  462,
			},
		},
	}
	contentMetadata := ContentMetadata{Height: 480, Width: 640}
	annotation := NewAnnotation("testid", "testid", "testid", "testid", []string{"testid"}, "", metatdata, contentMetadata)

	left, top, width, height := annotation.Metadata.BoundingBoxes[0].ToTopLeftWidthHeightFormat()

	if left != 98 || top != 345 || width != 322 || height != 117 {
		t.Fatalf("Expected left=%d top=%d width=%d height=%d; got left=%d top=%d width=%d height=%d",
			98, 345, 322, 117, left, top, width, height)
	}
}
