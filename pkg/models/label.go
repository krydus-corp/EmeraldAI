/*
 * File: labels.go
 * Project: upload
 * File Created: Wednesday, 22nd December 2021 1:15:55 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
)

/* Example labels.json
[
  { "tags": ["cat"], "external_id": "image1.jpeg" },
  { "tags": ["dog"], "external_id": "image2.jpeg" },
  { "tags": ["chicken", "cow"], "external_id": "image3.jpeg" }
]
*/

/* Example labels.json object detection
[
  { "tags": ["cat"], "external_id": "image1.jpeg", "bounding_boxes": [{"tagid": "cat", "xmin": 33, "ymin": 33, "xmax": 66, "ymax": 66}]},
  { "tags": ["dog"], "external_id": "image2.jpeg", "bounding_boxes": [{"tagid": "dog", "xmin": 33, "ymin": 33, "xmax": 66, "ymax": 66}]},
  { "tags": ["chicken", "cow"], "external_id": "image3.jpeg", "bounding_boxes": [{"tagid": "chicken", "xmin": 33, "ymin": 33, "xmax": 66, "ymax": 66}, {"tagid": "cow", "xmin": 133, "ymin": 133, "xmax": 166, "ymax": 166}]}
]
*/

// Label represents label domain model
//
// swagger:model Label
type Label struct {
	// Name of Tag
	//
	Tags []string `json:"tags"`
	//
	//
	ExternalID string `json:"external_id"`
	//
	//
	InternalID string `json:"internal_id"`
	//
	//
	Metadata []AnnotationDataBoundingBox `json:"bounding_boxes"`
}

type Labels []Label
type LabelMap map[string]Label

func (l *Label) Validate() error {
	for _, l := range l.Tags {
		if l == "" {
			return fmt.Errorf("1 or more label tags is null")
		}
	}

	if l.ExternalID == "" {
		return fmt.Errorf("label external id is null")
	}

	return nil
}

func (l *Label) Normalize() *Label {
	for i := 0; i < len(l.Tags); i++ {
		l.Tags[i] = strings.ToLower(l.Tags[i])
	}
	return l
}

// Check for labels file and create label-map if found
func ParseLabelsFromFile(labelsFile string, files []*multipart.FileHeader) (Labels, error) {
	labelSlice := []Label{}
	if labelsFile != "" {
		for _, fileHeader := range files {
			if fileHeader.Filename == labelsFile {
				file, err := fileHeader.Open()
				if err != nil {
					return nil, err
				}
				defer file.Close()

				fileBytes, err := io.ReadAll(file)
				if err != nil {
					return nil, err

				}
				if err := json.Unmarshal(fileBytes, &labelSlice); err != nil {
					return nil, err
				}
				break
			}
		}
	}
	return labelSlice, nil
}

// Validate labels and create label-map
func (l Labels) Validate(annotationType string) LabelMap {
	labelMap := make(map[string]Label)
	if len(l) > 0 {
		log.Debugf("creating label-map")
		for i := 0; i < len(l); i++ {
			if err := l[i].validateLabelType(annotationType); err != nil {
				log.Warnf("error validating label type; err=%s", err.Error())
				continue
			}
			if err := l[i].Validate(); err != nil {
				log.Warnf("error validating label; err=%s", err.Error())
				continue
			}
			// Create a mapping of the label to both the internal and external IDs.
			// For user created label files, either ExternalID or InternalID can be specified.
			// For a Emerald generated label file, the content is exported and named via the internal ID.
			// We export the content using internal IDs in the event that a file with the same name
			// of a file already uploaded to the project is uploaded.
			labelMap[l[i].ExternalID] = *l[i].Normalize()
			labelMap[l[i].InternalID] = *l[i].Normalize()
		}
	}

	return labelMap
}

func (l Label) validateLabelType(t string) error {
	if t == ProjectAnnotationTypeBoundingBox.String() {
		if l.Metadata == nil {
			log.Errorf("Unexpected format for bounding box project; label=%+v", l)
			return errors.New("invalid label format; label does not contain bounding box metadata")
		} else {
			return nil
		}
	} else if t == ProjectAnnotationTypeClassification.String() {
		if len(l.Metadata) > 0 {
			log.Errorf("Unexpected format for classification project; label=%+v", l)
			return errors.New("invalid label format; bounding box metadata found in classification project")
		} else {
			return nil
		}
	} else {
		return errors.New("invalid annotation type; unknown annotation type")
	}
}
