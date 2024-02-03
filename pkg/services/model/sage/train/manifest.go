/*
 * File: manifest.go
 * Project: train
 * File Created: Saturday, 20th August 2022 10:56:10 am
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package train

import (
	"encoding/json"
)

type ClassificationManifest struct {
	SourceRef string `json:"source-ref"`
	Class     string `json:"class"`
}

func NewClassificationManifest(source, class string) ClassificationManifest {
	return ClassificationManifest{source, class}
}

func (c ClassificationManifest) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

type ObjectDetectionManifest struct {
	SourceRef   string `json:"source-ref"`
	BoundingBox struct {
		ImageSize []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
			Depth  int `json:"depth"`
		} `json:"image_size"`
		Annotations []struct {
			ClassID int     `json:"class_id"`
			Left    float32 `json:"left"`
			Top     float32 `json:"top"`
			Width   float32 `json:"width"`
			Height  float32 `json:"height"`
		} `json:"annotations"`
		BoundingBoxMetadata struct {
			ClassMap map[int]string `json:"class_map"` // e.g.{ "17": "017.Cardinal" }
			Type     string         `json:"type"`      // e.g. "groundtruth/object-detection"
		} `json:"bounding-box-metadata"`
	} `json:"bounding-box"`
}

// NewObjectDetectionManifest creates a new object detection manifest.
//
// classMap: map of class int to name e.g. { "17": "017.Cardinal" }
//
// imageSizes: slice of image size in the format [width, height, depth]
//
// annotations: slice of annotation in the format [class_id, left, top, width, height]
// TODO - pass in objects for image size and annotations rather than assuming the correct info is at specific indices in these slices
func NewObjectDetectionManifest(source string, classMap map[int]string, imageSize []int, annotations [][]float32) ObjectDetectionManifest {
	manifest := ObjectDetectionManifest{
		SourceRef: source,
		BoundingBox: struct {
			ImageSize []struct {
				Width  int "json:\"width\""
				Height int "json:\"height\""
				Depth  int "json:\"depth\""
			} "json:\"image_size\""
			Annotations []struct {
				ClassID int     "json:\"class_id\""
				Left    float32 "json:\"left\""
				Top     float32 "json:\"top\""
				Width   float32 "json:\"width\""
				Height  float32 "json:\"height\""
			} "json:\"annotations\""
			BoundingBoxMetadata struct {
				ClassMap map[int]string "json:\"class_map\""
				Type     string         "json:\"type\""
			} "json:\"bounding-box-metadata\""
		}{
			Annotations: make([]struct {
				ClassID int     "json:\"class_id\""
				Left    float32 "json:\"left\""
				Top     float32 "json:\"top\""
				Width   float32 "json:\"width\""
				Height  float32 "json:\"height\""
			}, 0),
			BoundingBoxMetadata: struct {
				ClassMap map[int]string "json:\"class_map\""
				Type     string         "json:\"type\""
			}{
				ClassMap: classMap,
				Type:     "ObjectDetection",
			},
		},
	}

	manifest.BoundingBox.ImageSize = append(manifest.BoundingBox.ImageSize, struct {
		Width  int "json:\"width\""
		Height int "json:\"height\""
		Depth  int "json:\"depth\""
	}{
		Width:  imageSize[0],
		Height: imageSize[1],
		Depth:  imageSize[2],
	})

	for _, annotation := range annotations {
		if len(annotation) != 5 {
			// Null annotation
			continue
		}

		manifest.BoundingBox.Annotations = append(manifest.BoundingBox.Annotations, struct {
			ClassID int     "json:\"class_id\""
			Left    float32 "json:\"left\""
			Top     float32 "json:\"top\""
			Width   float32 "json:\"width\""
			Height  float32 "json:\"height\""
		}{
			ClassID: int(annotation[0]),
			Left:    annotation[1],
			Top:     annotation[2],
			Width:   annotation[3],
			Height:  annotation[4],
		})
	}
	return manifest
}

func (o ObjectDetectionManifest) ToJSON() ([]byte, error) {
	return json.Marshal(o)
}
