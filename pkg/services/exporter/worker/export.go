/*
 * File: metadata.go
 * Project: meta
 * File Created: Friday, 12th November 2021 4:27:15 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"

	common "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	zipwriter "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/archive"
	database "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
	annotationAPI "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/api/annotation"
)

const (
	// Default batch size for exports i.e. chunk size of export
	ExportBatchSize = 100000
	// Label file name
	LabelsFile = "labels.json"
)

type Export struct {
	Db       *database.DB
	Platform *platform.Platform
	Zipw     *zipwriter.ZipWriter

	Bucket string
}

func NewExport(db *database.DB, platform *platform.Platform, zipw *zipwriter.ZipWriter, bucket string) *Export {

	// Create uploader
	return &Export{
		Db:       db,
		Platform: platform,
		Zipw:     zipw,

		Bucket: bucket,
	}
}

func (e *Export) Export(export *models.Export) error {

	// Export
	switch export.Type {
	case models.ExportTypeProject:
		if err := e.generateProjectExport(
			e.Platform,
			export,
			ExportBatchSize); err != nil {
			log.Printf("error generating project export; userid=%s exportid=%s err=%s", export.UserID, export.ID.Hex(), err.Error())
			return err
		}
	case models.ExportTypeDataset:
		if err := e.generateDatasetExport(
			e.Platform,
			export); err != nil {
			log.Printf("error generating dataset export; userid=%s exportid=%s err=%s", export.UserID, export.ID.Hex(), err.Error())
			return err
		}
	case models.ExportTypeModel:
		return http.ErrNotSupported
	default:
		err := errors.New("Export requires project_id, dataset_id or model_id.")
		log.Printf("error generating export; userid=%s exportid=%s err=%s", export.UserID, export.ID.Hex(), err.Error())
		return err
	}

	return nil
}

// A method for batching and generating the content export and items list
func (e *Export) generateProjectExport(
	plat *platform.Platform,
	export *models.Export,
	batchSize int) error {

	opts := options.FindOptions{BatchSize: common.Ptr(int32(1000))}

	if export.Project == nil {
		return errors.Wrap(err, "unable to locate metadata for project export")
	}

	// Retrieve all content
	cursor, err := plat.ContentDB.FindProjectContent(e.Db, export.UserID, export.Project.ID.Hex(), &opts)
	if err != nil {
		log.Printf("error retrieving content metadata from database; export=%s err=%s", export.ID.Hex(), err.Error())
		return err
	}
	defer cursor.Close(context.TODO())

	// Iterate through all content
	log.Printf("iterating through content")

	batches := 0
	currentBatch := []Content{}

	for cursor.Next(context.TODO()) {
		var content models.Content
		if err = cursor.Decode(&content); err != nil {
			log.Printf("error decoding content metadata from database; export=%s err=%s", export.ID.Hex(), err.Error())
			return errors.Wrapf(err, "error decoding content from DB")
		}

		// Batch content
		currentBatch = append(currentBatch, Content{
			S3Bucket:         content.StoredDir,
			S3Key:            content.StoredPath,
			Archive:          fmt.Sprintf("%02d.zip", batches),
			Filename:         path.Base(content.StoredPath),
			FilenameOriginal: content.Name,
			MimeType:         content.ContentType,
			Size:             content.Size,
		})

		// Generate batch
		if len(currentBatch) == batchSize {
			archiveName := path.Join(export.Path, fmt.Sprintf("%02d.zip", batches))
			if err := e.uploadContentBatch(currentBatch, archiveName); err != nil {
				log.Printf("error processing export batch; export=%s err=%s", export.ID.Hex(), err.Error())
				return err
			}
			export.ContentKeys = append(export.ContentKeys, archiveName)
			// Reinitialize batch
			currentBatch = []Content{}

			batches++
		}
	}

	// Final batch
	if len(currentBatch) > 0 {
		archiveName := path.Join(export.Path, fmt.Sprintf("%02d.zip", batches))
		if err := e.uploadContentBatch(currentBatch, archiveName); err != nil {
			log.Printf("error processing export batch; export=%s err=%s", export.ID.Hex(), err.Error())
			return err
		}
		export.ContentKeys = append(export.ContentKeys, archiveName)
	}
	return nil
}

// uploadContentBatch is a method for generating content files for export
func (e *Export) uploadContentBatch(content []Content, archiveName string) error {

	// Upload archive
	inChan := make(chan *zipwriter.ObjectInput, 10)
	doneChan := make(chan error, 1)

	go func() {
		err := e.Zipw.ZipS3Files(inChan, &zipwriter.ObjectOutput{Bucket: &e.Bucket, Key: &archiveName})
		doneChan <- err
	}()

	// Upload content
	for i := 0; i < len(content); i++ {
		bucket := content[i].S3Bucket
		key := content[i].S3Key
		inChan <- &zipwriter.ObjectInput{Bucket: &bucket, Key: &key, RawBytes: nil}
	}

	// Upload metadata
	metaJSONBytes, err := ContentSlice(content).ToJSON()
	if err != nil {
		err = errors.Wrap(err, "error creating metadata file; unable to marshal to JSON")
		log.Println(err.Error())
		metaJSONBytes = []byte(fmt.Sprintf(`{"error": "%s"}`, err.Error()))
	}
	inChan <- &zipwriter.ObjectInput{RawBytes: map[string][]byte{"items.json": metaJSONBytes}}

	close(inChan)

	return <-doneChan
}

// A method for batching and generating the content export and items list
func (e *Export) generateDatasetExport(
	plat *platform.Platform,
	export *models.Export) error {

	// Annotations is a struct for modeling required info retrieved from the annotation DB
	type Annotation struct {
		ID        primitive.ObjectID        `bson:"_id"`
		TagIDs    []string                  `bson:"tagids"`
		ContentID string                    `bson:"contentid"`
		Metadata  models.AnnotationMetadata `bson:"metadata,omitempty"`
	}

	if export.Dataset == nil {
		return errors.Wrap(err, "unable to locate metadata for dataset export")
	}

	// Build labels file
	opts := options.FindOptions{Projection: bson.M{"_id": 1, "tagids": 1, "contentid": 1, "metadata": 1}, AllowDiskUse: common.Ptr(true)}
	cursor, err := plat.AnnotationDB.FindDatasetAnnotations(e.Db, export.UserID, export.Dataset.ID.Hex(), &opts)
	if err != nil {
		log.Printf("error retrieving dataset annotations from database; export=%s err=%s", export.ID.Hex(), err.Error())
		return err
	}
	defer cursor.Close(context.TODO())

	var annotations []*Annotation
	if err = cursor.All(context.TODO(), &annotations); err != nil {
		log.Printf("error decoding dataset annotations; export=%s err=%s", export.ID.Hex(), err.Error())
		return err
	}

	var labels = models.Labels{}
	tagCache := make(map[string]models.Tag) // maps tagid -> tag
	for _, annotation := range annotations {
		tags := []string{}
		for _, tagid := range annotation.TagIDs {
			// Fetch tag
			tag, ok := tagCache[tagid]
			if !ok {
				tag, err = e.Platform.TagDB.View(e.Db, export.UserID, tagid)
				if err != nil {
					return errors.Wrapf(err, "error retrieving tag from database; tag=%s annotation=%s", tagid, annotation.ID.Hex())
				}
				tagCache[tagid] = tag
			}
			tags = append(tags, tag.Name)
		}

		// Fetch content
		content, err := e.Platform.ContentDB.View(e.Db, export.UserID, annotation.ContentID)
		if err != nil {
			return errors.Wrapf(err, "error retrieving content from database; content=%s annotation=%s", annotation.ContentID, annotation.ID.Hex())
		}

		// Replace bounding-box tagID with name of tag
		for i := 0; i < len(annotation.Metadata.BoundingBoxes); i++ {
			id := annotation.Metadata.BoundingBoxes[i].TagID
			annotation.Metadata.BoundingBoxes[i].TagID = tagCache[id].Name
		}

		// build labels file
		labels = append(labels, models.Label{
			Tags:       tags,
			Metadata:   annotation.Metadata.BoundingBoxes,
			ExternalID: content.Name,
			InternalID: path.Base(content.StoredPath),
		})
	}

	// Marshal file
	labelsFile, err := json.Marshal(labels)
	if err != nil {
		log.Printf("error creating labels file; err=%s", err.Error())
		return err
	}

	// Upload archive
	inChan := make(chan *zipwriter.ObjectInput, 10)
	doneChan := make(chan error, 1)
	archiveName := path.Join(export.Path, LabelsFile+".zip")

	go func() {
		err := e.Zipw.ZipS3Files(inChan, &zipwriter.ObjectOutput{Bucket: &e.Bucket, Key: &archiveName})
		doneChan <- err
	}()

	// Upload file
	inChan <- &zipwriter.ObjectInput{RawBytes: map[string][]byte{LabelsFile: labelsFile}}

	close(inChan)

	export.ContentKeys = append(export.ContentKeys, archiveName)

	// Attach statistics
	metadata, err := annotationAPI.Initialize(e.Db, e.Platform, nil).Statistics(nil, export.UserID, export.Dataset.ProjectID, export.Dataset.ID.Hex(), nil)
	if err != nil {
		log.Printf("error fetching dataset statistics; err=%s", err.Error())
		return err
	}

	var metadataMap map[string]interface{}
	if err := mapstructure.Decode(metadata, &metadataMap); err != nil {
		log.Printf("error decoding dataset statistics; err=%s", err.Error())
		return err
	}

	export.Metadata = metadataMap

	return <-doneChan
}
