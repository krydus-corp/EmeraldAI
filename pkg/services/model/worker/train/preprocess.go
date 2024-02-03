/*
 * File: preprocess.go
 * Project: train
 * File Created: Tuesday, 16th August 2022 7:17:06 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package train

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/pkg/errors"

	common "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	train "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/sage/train"
)

func (w *WorkerPool) preprocess(model *models.Model, dataset *models.Dataset, project *models.Project, labelIntegerMap map[string]int) (*SplitCounts, error) {

	// Determine dataset split
	counts, err := GetSplitCounts(dataset, float64(dataset.Split.Train), float64(dataset.Split.Validation), float64(dataset.Split.Test), *w.Platform.AnnotationDB, w.DB)
	if err != nil {
		return nil, err
	}
	log.Debugf("computed train splits; train=%d validation=%d test=%d", counts.TrainCount, counts.ValidationCount, counts.TestCount)

	// Fetch all annotations associated with this dataset (only necessary data)
	annotations, err := FetchAnnotations(dataset, w.Platform, w.DB)
	if err != nil {
		return nil, err
	}

	// Randomly shuffle our data
	annotations.Shuffle()

	// Split annotation
	train, validation, test := annotations.Split(counts)
	if len(train) == 0 || len(validation) == 0 {
		return nil, ErrNoContent
	}
	log.Debugf("split annotations; train=%d validation=%d test=%d", train.Length(), validation.Length(), test.Length())

	// Create manifest files according to split
	trainManifest, trainErr := w.generateManifest(dataset, project, train, labelIntegerMap)
	validationManifest, validationErr := w.generateManifest(dataset, project, validation, labelIntegerMap)
	testManifest, testErr := w.generateManifest(dataset, project, test, labelIntegerMap)
	if err := common.CombineErrors([]error{trainErr, validationErr, testErr}); err != nil {
		return nil, fmt.Errorf("error occurred during manifest file generation; dataset=%s err=%s", dataset.ID.Hex(), err.Error())
	}

	// Upload manifest files!
	if len(trainManifest) > 0 {
		key := fmt.Sprintf("%s/models/%s/train.manifest", dataset.UserID, model.ID.Hex())
		location, err := w.Blob.Uploader.Upload(bytes.NewReader(trainManifest), w.Blob.Bucket, key)
		if err != nil {
			return nil, fmt.Errorf("error occurred uploading manifest file; dataset=%s err=%s", dataset.ID.Hex(), err.Error())
		}
		log.Debugf("uploaded train manifest to: %s (%d bytes)", location, len(trainManifest))
	}
	if len(validationManifest) > 0 {
		key := fmt.Sprintf("%s/models/%s/validation.manifest", dataset.UserID, model.ID.Hex())
		location, err := w.Blob.Uploader.Upload(bytes.NewReader(validationManifest), w.Blob.Bucket, key)
		if err != nil {
			return nil, fmt.Errorf("error occurred uploading manifest file; dataset=%s err=%s", dataset.ID.Hex(), err.Error())
		}
		log.Debugf("uploaded validation manifest to: %s (%d bytes)", location, len(validationManifest))
	}
	if len(testManifest) > 0 {
		key := fmt.Sprintf("%s/models/%s/test.manifest", dataset.UserID, model.ID.Hex())
		location, err := w.Blob.Uploader.Upload(bytes.NewReader(testManifest), w.Blob.Bucket, key)
		if err != nil {
			return nil, fmt.Errorf("error occurred uploading manifest file; dataset=%s err=%s", dataset.ID.Hex(), err.Error())
		}
		log.Debugf("uploaded test manifest to: %s (%d bytes)", location, len(testManifest))
	}

	// Update annotation splits
	log.Debugf("updating annotation splits...")
	updated := 0
	for split, data := range map[models.Split]interface{}{models.SplitTrain: train, models.SplitValidation: validation, models.SplitTest: test} {
		data := data.(Annotations)
		for i := 0; i < len(data); i++ {
			annotation := data[i]
			update := &models.Annotation{
				ID:     annotation.ID,
				UserID: dataset.UserID,
				Split:  split.String(),
				TagIDs: nil, // Don't update!
			}
			updated++
			if err := w.Platform.AnnotationDB.Update(w.DB, update); err != nil {
				return nil, err
			}
		}
	}
	log.Debugf("updated %d anotations", updated)

	return &counts, nil
}

func (w *WorkerPool) generateManifest(dataset *models.Dataset, project *models.Project, annotations Annotations, labelIntegerMap map[string]int) ([]byte, error) {
	tagCache := make(map[string]models.Tag) // maps tagid -> tag
	writer := bytes.NewBufferString("")

	log.Debugf("generating manifest for %d annotations", len(annotations))

loop:
	for i := 0; i < len(annotations); i++ {
		var content *models.Content

		// Fetch content associated with annotation
		content, err := w.Platform.ContentDB.View(w.DB, dataset.UserID, annotations[i].ContentID)
		if err != nil {
			return nil, errors.Wrapf(err, "error retrieving content from database; user=%s content=%s", dataset.UserID, annotations[i].ContentID)
		}

		s3Path := "s3://" + path.Join(content.StoredDir, content.StoredPath)

		// Create an entry for each content-tag pair
		if len(annotations[i].TagIDs) == 0 { // Nil annotation
			if project.AnnotationType == models.ProjectAnnotationTypeClassification.String() {
				labels := classLabels(len(labelIntegerMap), make(map[int]struct{}))
				entry, err := train.NewClassificationManifest(s3Path, labels).ToJSON()
				if err != nil {
					return nil, err
				}
				if _, err := writer.WriteString(string(entry) + "\n"); err != nil {
					return nil, err
				}
			} else {
				imageSize := []int{content.Width, content.Height, 3}
				entry, err := train.NewObjectDetectionManifest(s3Path, common.ReverseMapStringInt(labelIntegerMap), imageSize, [][]float32{}).ToJSON()
				if err != nil {
					return nil, err
				}
				if _, err := writer.WriteString(string(entry) + "\n"); err != nil {
					return nil, err
				}
			}
			continue loop
		}

		labelIndices := make(map[int]struct{})
		boundingBoxes := [][]float32{}

		for j := 0; j < len(annotations[i].TagIDs); j++ {
			tagid := annotations[i].TagIDs[j]

			// Fetch tag
			tag, ok := tagCache[tagid]
			if !ok {
				tag, err = w.Platform.TagDB.View(w.DB, dataset.UserID, tagid)
				if err != nil {
					return nil, errors.Wrapf(err, "error retrieving tag from database; tag=%s annotation=%s", tagid, annotations[i].ID.Hex())
				}
				tagCache[tagid] = tag
			}

			idx, ok := labelIntegerMap[tag.Name]
			if !ok {
				l, _ := json.Marshal(labelIntegerMap) // intentionally ignored
				return nil, errors.Wrapf(err, "error looking up content's tag name against current label map; tag=%s, label-map=%s", tagid, string(l))
			}

			labelIndices[idx] = struct{}{}
			metadata := annotationMetadata(annotations[i], idx, tagid)
			boundingBoxes = append(boundingBoxes, metadata...)
		}

		if project.AnnotationType == models.ProjectAnnotationTypeClassification.String() {
			labels := classLabels(len(labelIntegerMap), labelIndices)
			entry, err := train.NewClassificationManifest(s3Path, labels).ToJSON()
			if err != nil {
				return nil, err
			}
			if _, err := writer.WriteString(string(entry) + "\n"); err != nil {
				return nil, err
			}
		} else {
			imageSize := []int{content.Width, content.Height, 3}
			entry, err := train.NewObjectDetectionManifest(s3Path, common.ReverseMapStringInt(labelIntegerMap), imageSize, boundingBoxes).ToJSON()
			if err != nil {
				return nil, err
			}
			if _, err := writer.WriteString(string(entry) + "\n"); err != nil {
				return nil, err
			}
		}

	}
	return writer.Bytes(), nil
}

// annotationMetadata is a helper function for retrieving and formatting an annotations bounding box metadata
func annotationMetadata(annotation *Annotation, classID int, tagid string) (formattedBoundingBoxes [][]float32) {
	for _, boundingBox := range annotation.Metadata.BoundingBoxes {
		if boundingBox.TagID == tagid {
			left, top, width, height := boundingBox.ToTopLeftWidthHeightFormat()
			formattedBoundingBoxes = append(formattedBoundingBoxes, []float32{float32(classID), float32(left), float32(top), float32(width), float32(height)})
		}
	}
	return
}

// classLabels is a helper function for creating a multi-hot formatted string of labels
// If for example, there are 10 classes and the image is labeld for the the first and 5th class -> classLabels(10, map[int]struct{}{0: {}, 5: {}}
func classLabels(len int, labels map[int]struct{}) string {
	a := make([]string, len)
	for i := 0; i < len; i++ {
		if _, ok := labels[i]; ok {
			a[i] = "1"
		} else {
			a[i] = "0"
		}
	}

	return "[" + strings.Join(a, ",") + "]"
}
