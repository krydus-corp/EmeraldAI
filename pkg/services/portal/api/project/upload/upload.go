/*
 * File: go
 * Project: upload
 * File Created: Wednesday, 4th January 2023 7:47:40 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package upload

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strings"
	"time"

	blob "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/blob"
	common "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	image "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/image"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

type UploadArgs struct {
	File     *multipart.FileHeader
	Blob     *blob.Blob
	Platform *platform.Platform
	DB       *db.DB
	Report   *Report
	Project  *models.Project

	LabelMap models.LabelMap

	UploadMaxFileSizeMB int
	UploadMaxImageWidth int
	UploadImageFormat   string
}

func (a *UploadArgs) Upload() (_ struct{}, _ error) {
	a.Report.TotalBytes += a.File.Size

	// Skip labels file
	if strings.HasSuffix(a.File.Filename, "json") {
		a.Report.TotalImagesSucceeded++
		return
	}

	// Restrict the size of each uploaded file to 5MB.
	if int(a.File.Size) > a.UploadMaxFileSizeMB {
		log.Errorf("max file exceeded; file=%s", a.File.Filename)
		a.Report.AddErr(nil, UploadErrMaxSizeExceeded, a.File.Filename, fmt.Sprintf("Image must be <= %dMB", a.UploadMaxFileSizeMB/(1024*1024)))
		return
	}

	// Open the file
	file, err := a.File.Open()
	if err != nil {
		log.Errorf("error opening file; file=%s, err=%s", a.File.Filename, err.Error())
		a.Report.AddErr(err, UploadErrIO, a.File.Filename, "Unable to open file")
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Errorf("error reading file; file=%s, err=%s", a.File.Filename, err.Error())
		a.Report.AddErr(err, UploadErrIO, a.File.Filename, "Unable to read file")
		return
	}
	file.Close()

	// Check file type
	filetype := http.DetectContentType(fileBytes)
	if filetype != "image/jpeg" && filetype != "image/png" {
		log.Errorf("error detecting content type; file=%s, err=%s", a.File.Filename, err.Error())
		a.Report.AddErr(nil, UploadErrInvalidFormat, a.File.Filename, "The provided file format is not allowed. Please upload a JPEG or PNG image")
		return
	}

	// Get file stats
	stats, err := image.GetStats(fileBytes)
	if err != nil {
		log.Errorf("error getting image stats; file=%s, err=%s", a.File.Filename, err.Error())
		a.Report.AddErr(err, UploadErrStats, a.File.Filename, "Unable to get file stats")
		return
	}

	// Hash, create thumbnail, resize, and format
	imagehash, thumb, err := ProcessImage(&File{FileName: a.File.Filename, FileBytes: fileBytes}, stats, a.UploadMaxImageWidth, a.UploadImageFormat)
	if err != nil {
		log.Errorf("error processing image; file=%s, err=%s", a.File.Filename, err.Error())
		a.Report.AddErr(err, UploadErrProcessing, a.File.Filename, "Unable to process image")
		return
	}

	// Upload to blob store
	newKey := fmt.Sprintf("%s/content/%s.%s", a.Project.UserID, imagehash, path.Base(a.UploadImageFormat))
	_, err = a.Blob.Uploader.Upload(bytes.NewBuffer(fileBytes), a.Blob.Bucket, newKey)
	if err != nil {
		log.Errorf("error uploading to blob store; file=%s, err=%s", a.File.Filename, err.Error())
		a.Report.AddErr(err, UploadErrProcessing, a.File.Filename, "Unable to process image")
		return
	}

	// Create new content
	contentID := models.NewContentID(imagehash, a.Project.UserID)
	content, err := a.Platform.ContentDB.View(a.DB, a.Project.UserID, contentID)
	if err != nil {
		if err == platform.ErrContentDoesNotExist {
			// Create new content
			newContent := models.NewContent(
				a.UploadImageFormat,
				a.Blob.Bucket,
				newKey,
				imagehash,
				a.Project.UserID,
				a.Project.ID.Hex(),
				path.Base(a.File.Filename),
				thumb,
				len(fileBytes),
				stats.Height,
				stats.Width)

			if _, err := a.Platform.ContentDB.Create(a.DB, newContent); err != nil {
				log.Errorf("unable to create content metadata; file=%s, err=%s", a.File.Filename, err.Error())
				if err := a.Blob.S3Client.DeleteObject(a.Blob.Bucket, newKey); err != nil {
					log.Errorf("error removing content blob after metadata creation failure; file=%s, err=%s", a.File.Filename, err.Error())
				}
				a.Report.AddErr(err, UploadErrProcessing, a.File.Filename, "Unable to process image")
				return
			}

			a.Report.TotalImagesSucceeded++

			// Annotate content if found in included labelmap
			if err := AnnotateContent(a.DB, a.Platform, newContent, fileBytes, a.LabelMap, a.Project.AnnotationType, a.Project.DatasetID, a.Project.ID.Hex(), a.File.Filename); err != nil {
				log.Errorf("error annotating content; id=%s, file=%s err=%s", contentID, a.File.Filename, err.Error())
				a.Report.AddErr(err, UploadErrAnnotation, a.File.Filename, "Unable to annotate image")
				return
			}

		} else {
			log.Errorf("error looking up content; id=%s, file=%s err=%s", contentID, a.File.Filename, err.Error())
			a.Report.AddErr(err, UploadErrProcessing, a.File.Filename, "Unable to process image")
			return
		}
	} else {
		// Check if content already associated with project
		// TODO: This logic prevents content that has already been uploaded (to any project!) from being annotated again.
		// TODO: User needs to be able to replace the labels for content uploaded with a labels file.

		if common.Contains(content.Projects, a.Project.ID.Hex()) {
			log.Debugf("content already associated with project; id=%s, project=%s (%s) content=%s", a.Project.UserID, a.Project.ID.Hex(), a.Project.Name, contentID)
			a.Report.TotalFilesDuplicate++
			return
		}

		// Update content with new project association
		content.Projects = append(content.Projects, a.Project.ID.Hex())
		content.UpdatedAt = time.Now()
		if err := a.Platform.ContentDB.Update(a.DB, content); err != nil {
			log.Errorf("unable to update content; content=%s, err=%s", content.ID, err.Error())
			a.Report.AddErr(err, UploadErrProcessing, a.File.Filename, "Unable to process image")
			return
		}

		a.Report.TotalImagesSucceeded++

		// Label content if found in included labelmap
		if err := AnnotateContent(a.DB, a.Platform, content, fileBytes, a.LabelMap, a.Project.AnnotationType, a.Project.DatasetID, a.Project.ID.Hex(), a.File.Filename); err != nil {
			log.Errorf("failed annotating content; content=%s, err=%s", content.ID, err.Error())
			a.Report.AddErr(err, UploadErrAnnotation, a.File.Filename, "Unable to annotate image")
			return
		}
	}

	return
}
