/*
 * File: process.go
 * Project: upload
 * File Created: Wednesday, 4th January 2023 12:29:37 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package upload

import (
	"path"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	image "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/image"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

type File struct {
	FileBytes []byte
	FileName  string
}

// annotateContent is a helper method for annotating content based on provided labelMap
func AnnotateContent(db *db.DB, plat *platform.Platform, content *models.Content, contentBytes []byte, labelMap models.LabelMap, projectAnnotationType, datasetid, projectid, filename string) error {
	l, ok := labelMap[path.Base(filename)]
	if !ok {
		// File does not have tag association
		log.Debugf("File does not have tag association; filename=%s", filename)

		return nil
	}

	if len(l.Tags) == 0 {
		log.Debugf("No tags; filename=%s label=%+v", filename, l)
		return nil
	}

	// Create tags
	tagids := []string{}
	tagmap := make(map[string]string)
	for _, name := range l.Tags {
		tag, err := plat.TagDB.FindByName(db, content.UserID, datasetid, name)
		if err != nil {
			if err == platform.ErrTagDoesNotExist {
				// Create the tag
				log.Debugf("creating tag: name=%s", name)
				tagModel := models.NewTag(content.UserID, projectid, datasetid, name, []string{})
				tag, err = plat.TagDB.Create(db, tagModel)
				if err != nil {
					// Even though we check for ErrTagDoesNotExist, we need to check again b/c this Op is not thread safe i.e.
					// it could have been created by another goroutine in the time after checking for ErrTagDoesNotExist and before creating.
					if !errors.Is(err, platform.ErrTagAlreadyExists) {
						return errors.Wrapf(err, "unexpected error creating tag associated with name=%s", name)
					}
				}
			} else {
				return errors.Wrapf(err, "unexpected error retrieving tag associated with name=%s", name)
			}
		}

		tagmap[name] = tag.ID.Hex()
		tagids = append(tagids, tag.ID.Hex())
	}

	meta := models.AnnotationMetadata{}
	imgBase64 := ""
	if projectAnnotationType == models.ProjectAnnotationTypeBoundingBox.String() {
		boundingBoxes := []image.BoundingBox{}
		for _, bbox := range l.Metadata {
			// AnnotationDataBoundingBox.TagID is the tag name in the context of a labels file import
			meta.BoundingBoxes = append(meta.BoundingBoxes, models.AnnotationDataBoundingBox{
				TagID: tagmap[bbox.TagID],
				Xmin:  bbox.Xmin,
				Ymin:  bbox.Ymin,
				Xmax:  bbox.Xmax,
				Ymax:  bbox.Ymax,
			})
			boundingBoxes = append(boundingBoxes, image.BoundingBox{
				Xmin:      bbox.Xmin,
				Xmax:      bbox.Xmax,
				Ymin:      bbox.Ymin,
				Ymax:      bbox.Ymax,
				ClassName: bbox.TagID,
			})
		}

		// Update Base64 img.
		thumb, _, err := image.ThumbnailBoundingBox(contentBytes,
			100,
			100,
			boundingBoxes)
		if err != nil {
			return errors.Wrapf(err, "error creating annotation thumbnail for content=%s", content.ID)
		}
		imgBase64 = thumb
	}

	// Create annotation models
	annotation := models.NewAnnotation(
		content.UserID,
		projectid,
		datasetid,
		content.ID,
		tagids,
		imgBase64,
		meta,
		models.ContentMetadata{Size: content.Size, Height: content.Height, Width: content.Width})
	// Validate
	if err := annotation.Valid(projectAnnotationType); err != nil {
		return err
	}
	// Create annotation at DB
	_, err := plat.AnnotationDB.Create(db, *annotation)
	if err != nil && !errors.Is(err, platform.ErrAnnotationAlreadyExists) {
		return err
	}

	return nil
}

// processImage is a helper function for getting image hash and create thumbnail (these can be done in parallel)
func ProcessImage(f *File, stats *image.Stats, maxImageWidth int, imageFmt string) (imagehash, thumb string, err error) {
	var g errgroup.Group

	// Check max dimensions
	// Cannot parallelize as this modifies the image
	if stats.Width > maxImageWidth {
		resized, err := image.ResizeImage(f.FileBytes, maxImageWidth, 0)
		if err != nil {
			return "", "", errors.Wrapf(err, "unable to resize image; file=%s", f.FileName)
		}
		f.FileBytes = resized
	}

	// Convert image type
	// Cannot parallelize as this modifies the image
	if stats.ContentType != imageFmt {
		converted, err := image.ConvertImgType(f.FileBytes, imageFmt)
		if err != nil {
			return "", "", errors.Wrapf(err, "unable to format image; file=%s", f.FileName)
		}
		f.FileBytes = converted
	}

	g.Go(func() error {
		imagehash, err = image.Hash(f.FileBytes)
		if err != nil {
			return errors.Wrapf(err, "unable to hash image; file=%s", f.FileName)
		}
		return nil
	})

	// Create thumbnail
	g.Go(func() error {
		thumb, _, err = image.Thumbnail(f.FileBytes, 100, 100)
		if err != nil {
			return errors.Wrapf(err, "unable to create thumbnail from image; file=%s", f.FileName)
		}
		return nil
	})

	err = g.Wait()

	return
}
