// Package project contains project application services
package project

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/sync/errgroup"

	image "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/image"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	worker "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/worker"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	garbageBL "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/worker/garbage"
	upload "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/api/project/upload"
)

var ErrProjectEmpty = echo.NewHTTPError(http.StatusInternalServerError, "Project is empty")

const (
	UploadConcurrency      = 64
	UploadRetries          = 5
	UploadRetryWaitSeconds = 2
	UploadBackoff          = true
	UploadPoolThreshold    = 10

	// 3MB to align with Sagemaker serverless inference size limit (4MB) and provide a buffer for other request content.
	// https://docs.aws.amazon.com/sagemaker/latest/dg/serverless-endpoints-invoke.html
	UploadMaxFileSizeMB = 1024 * 1024 * 3
	UploadMaxImageWidth = 1280
	UploadImageFormat   = "image/jpeg"
)

// Create is a method for creating a new project entry
func (p Project) Create(c echo.Context, projectModel *models.Project, datasetModel *models.Dataset) (models.Project, error) {
	// Create dataset
	if _, err := p.platform.DatasetDB.Create(p.db, datasetModel); err != nil {
		return models.Project{}, fmt.Errorf("error creating dataset; err=%s", err.Error())
	}

	// Create project
	project, err := p.platform.ProjectDB.Create(p.db, *projectModel)
	if err != nil {
		return models.Project{}, err
	}

	return project, nil
}

// List is a method for listing all projects for the given user
func (p Project) List(c echo.Context, userid string, page models.Pagination) ([]models.Project, int64, error) {
	return p.platform.ProjectDB.List(p.db, userid, page)
}

// View is a method for returning a single project by it's ID
func (p Project) View(c echo.Context, userid, projectid string) (models.Project, error) {
	project, err := p.platform.ProjectDB.View(p.db, userid, projectid)
	if err != nil {
		return models.Project{}, err
	}

	return project, nil
}

func (p Project) Query(ctx echo.Context, userid string, q models.Query) ([]models.Project, int64, error) {
	return p.platform.ProjectDB.Query(p.db, userid, q)
}

// Delete is a method for removing project data form the database along with any associated content.
// Content is removed via a garbage collector.
func (p Project) Delete(c echo.Context, userid, projectid string) error {

	// Delete project
	if err := p.platform.ProjectDB.Delete(p.db, userid, projectid); err != nil {
		return err
	}

	// Remove project associations
	if err := p.platform.ContentDB.PullProjectAssociation(p.db, userid, projectid); err != nil {
		return err
	}

	// Delete any Content no longer associated with any projects
	cursor, err := p.platform.ContentDB.FindOrphanedContent(p.db, userid)
	if err != nil {
		return err
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var content models.Content
		if err = cursor.Decode(&content); err != nil {
			log.Errorf("error retrieving content metadata from database; user=%s project=%s err=%s", userid, projectid, err.Error())
			continue
		}
		// Send to blob delete queue
		p.blob.DeleteChan <- content.StoredPath

		// Delete metadata
		if err := p.platform.ContentDB.Delete(p.db, content.ID); err != nil {
			log.Errorf("error removing content metadata from database; user=%s project=%s err=%s", userid, projectid, err.Error())
			continue
		}
	}

	// Delete any Tags associated with the project
	if err := p.platform.TagDB.DeleteInProject(p.db, userid, projectid); err != nil {
		return err
	}

	// Delete any Models (metadata) associated with the project
	cursor, err = p.platform.ModelDB.FindProjectModels(p.db, userid, projectid)
	if err != nil {
		return err
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var model models.Model
		if err = cursor.Decode(&model); err != nil {
			log.Errorf("error retrieving model metadata from database; user=%s project=%s err=%s", userid, projectid, err.Error())
			continue
		}
		if err := p.platform.ModelDB.DeleteMany(p.db, userid, model.ID); err != nil {
			log.Errorf("error deleting model metadata from database; user=%s project=%s err=%s", userid, projectid, err.Error())
			continue
		}

		// Cleanup model resources if necessary.
		if model.Deployment.ModelName != "" && model.Deployment.EndpointName != "" {
			// Queue up model delete job
			if _, err := worker.Retry(3, true, true, 2, func() (struct{}, error) {
				return struct{}{}, p.garbagePublisher.Publish(context.TODO(), garbageBL.NewEvent(model.ID.Hex(), model.Deployment.ModelName, model.Deployment.EndpointName, userid, garbageBL.ActionDelete))
			}); err != nil {
				log.Errorf("error queuing up model delete job; model=%s error=%s", model.ID.Hex(), err.Error())
				return err
			}
		}

		// Delete any Models (blob) associated with the project
		p.blob.DeleteChan <- model.Key()
	}

	// Delete any Datasets associated with the project
	if err := p.platform.DatasetDB.DeleteProjectDatasets(p.db, userid, projectid); err != nil {
		return err
	}

	// Delete any Annotations associated with the project
	if err := p.platform.AnnotationDB.DeleteProjectAnnotations(p.db, userid, projectid); err != nil {
		return err
	}

	// Delete any Exports associated with the project
	cursor, err = p.platform.ExportDB.FindProjectExports(p.db, userid, projectid)
	if err != nil {
		return err
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var export models.Export
		if err = cursor.Decode(&export); err != nil {
			log.Errorf("error retrieving export metadata from database; user=%s project=%s err=%s", userid, projectid, err.Error())
			continue
		}
		// Send to blob delete queue
		p.blob.DeleteChan <- export.Path

		// Delete metadata
		if err := p.platform.ExportDB.Delete(p.db, export.ID); err != nil {
			log.Errorf("error removing export metadata from database; user=%s project=%s err=%s", userid, projectid, err.Error())
			continue
		}
	}

	return nil
}

// Profile is a method for uploading a file to a project that is used for the project profile picture or avatar.
// If the passed in *multipart.File is nil, a random image from the project is chosen.
func (p Project) Profile(c echo.Context, userid, projectid, filename string, file *multipart.File) (string, error) {
	var fileBytes []byte
	var err error

	project, err := p.platform.ProjectDB.View(p.db, userid, projectid)
	if err != nil {
		return filename, err
	}

	if file != nil {
		// User defiled image
		fileBytes, err = io.ReadAll(*file)
		if err != nil {
			return filename, err
		}
	} else {
		// Random image
		if project.Count == 0 {
			return filename, ErrProjectEmpty
		}

		content, err := p.platform.ContentDB.Sample(p.db, userid, projectid, 1, false)
		if err != nil {
			return filename, err
		}

		contentBytes, err := p.blob.Get(content[0].StoredDir, content[0].StoredPath)
		if err != nil {
			return filename, err
		}

		fileBytes = contentBytes
		filename = content[0].Name
	}

	// Create base64 images
	stats, err := image.GetStats(fileBytes)
	if err != nil {
		return filename, err
	}

	if !strings.Contains(stats.ContentType, "jpg") && !strings.Contains(stats.ContentType, "jpeg") && !strings.Contains(stats.ContentType, "png") {
		return filename, fmt.Errorf("uploaded content must be an image (.jpeg, .jpg, or .png)")
	}

	// Create thumbnail
	var (
		g                            errgroup.Group
		thumb100, thumb200, thumb640 string
	)

	g.Go(func() error {
		thumb100, _, err = image.Thumbnail(fileBytes, 100, 100)
		if err != nil {
			return errors.Wrapf(err, "unable to create thumbnail from image; file=%s", filename)
		}
		return nil
	})

	g.Go(func() error {
		thumb200, _, err = image.Thumbnail(fileBytes, 200, 200)
		if err != nil {
			return errors.Wrapf(err, "unable to create thumbnail from image; file=%s", filename)
		}
		return nil
	})

	g.Go(func() error {
		thumb640, _, err = image.Thumbnail(fileBytes, 640, 640)
		if err != nil {
			return errors.Wrapf(err, "unable to create thumbnail from image; file=%s", filename)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return filename, err
	}

	// Update project with avatars
	project.Profile100 = thumb100
	project.Profile200 = thumb200
	project.Profile640 = thumb640

	return filename, p.platform.ProjectDB.Update(p.db, project)
}

// Update is a struct which contains project information used for updating.
type Update struct {
	ProjectID   string
	UserID      string
	Name        string
	License     *string
	Description *string
}

// Update is a method for updating a project in project db.
func (p Project) Update(c echo.Context, r Update) (models.Project, error) {
	id, err := primitive.ObjectIDFromHex(r.ProjectID)
	if err != nil {
		return models.Project{}, err
	}

	if err := p.platform.ProjectDB.Update(p.db, models.Project{
		ID:          id,
		UserID:      r.UserID,
		Name:        r.Name,
		Description: r.Description,
		LicenseType: r.License,
	}); err != nil {
		return models.Project{}, err
	}

	return p.platform.ProjectDB.View(p.db, r.UserID, r.ProjectID)
}

type CreateUploadReq struct {
	UserID     string
	ProjectID  string
	LabelsFile string
	Files      []*multipart.FileHeader
}

// Create creates a new upload entry
func (p Project) Upload(c echo.Context, req CreateUploadReq) (*upload.Report, error) {
	// Check project exists
	project, err := p.platform.ProjectDB.View(p.db, req.UserID, req.ProjectID)
	if err != nil {
		return nil, err
	}

	// Check for labels file and create label-map if found
	now := time.Now()
	labelSlice, err := models.ParseLabelsFromFile(req.LabelsFile, req.Files)
	if err != nil {
		log.Warnf("unable to parse labels file=%s; skipping labeling. Error=%s", req.LabelsFile, err.Error())
	}
	log.Debugf("Parse labels time: %dms", time.Since(now).Milliseconds())

	// Validate labels
	labelMap := labelSlice.Validate(project.AnnotationType)

	// Upload metadata updates
	report := upload.NewReport(req.LabelsFile, len(req.Files))

	defer func() {
		go func() {
			// Update Project count
			if err := p.platform.ProjectDB.UpdateCount(p.db, req.ProjectID, req.UserID, report.TotalImagesSucceeded); err != nil {
				log.Errorf("unable to update project count; project=%s, err=%s", req.ProjectID, err.Error())
			}
			// Update usage
			if err := p.platform.UserDB.AddUsage(p.db, req.UserID, models.Usage{
				Time:          time.Now().UTC().Format(time.RFC3339),
				Type:          models.UsageTypeUpload,
				BillingMetric: models.BillingMetricGB,
				BillableValue: float64(report.TotalBytes) / 1_000_000_000,
				Metadata:      map[string]interface{}{"projectid": req.ProjectID, "object_count": len(req.Files)},
			}); err != nil {
				log.Errorf("Unable to update usage for project=%s user=%s; unable to locate user", req.ProjectID, req.UserID)
			}
		}()
	}()

	// Don't spin up the pool if less than threshold
	if len(req.Files) < UploadPoolThreshold {
		for _, fileHeader := range req.Files {
			args := upload.UploadArgs{
				File:                fileHeader,
				Blob:                p.blob,
				Platform:            p.platform,
				DB:                  p.db,
				Report:              report,
				Project:             &project,
				LabelMap:            labelMap,
				UploadMaxFileSizeMB: UploadMaxFileSizeMB,
				UploadMaxImageWidth: UploadMaxImageWidth,
				UploadImageFormat:   UploadImageFormat,
			}
			args.Upload()
		}
		return report, nil
	}

	pool, sent, received := worker.New[struct{}](worker.Config{
		Concurrency:      UploadConcurrency,
		RetryAttempts:    UploadRetries,
		RetryWaitSeconds: UploadRetryWaitSeconds,
		RetryBackoff:     UploadBackoff,
		RetryJitter:      true,
		Name:             fmt.Sprintf("upload=%s", uuid.New()),
	}).Start(), 0, 0

	go func() {
		log.Debugf("starting upload progress routine; project=%s time=%d", req.ProjectID, time.Now().UnixMilli())
		for range pool.OutChan {
			received++
		}
	}()

	// Feed the pool
	log.Debugf("starting upload feed loop; project=%s time=%s", req.ProjectID, time.Now().UnixMilli())
	for _, fileHeader := range req.Files {
		args := upload.UploadArgs{
			File:                fileHeader,
			Blob:                p.blob,
			Platform:            p.platform,
			DB:                  p.db,
			Report:              report,
			Project:             &project,
			LabelMap:            labelMap,
			UploadMaxFileSizeMB: UploadMaxFileSizeMB,
			UploadMaxImageWidth: UploadMaxImageWidth,
			UploadImageFormat:   UploadImageFormat,
		}
		pool.InChan <- args.Upload
		sent++
	}

	// Wait for completion
	for received != sent {
		log.Debugf("waiting for upload to finish; sent=%d received=%d", sent, received)
		time.Sleep(1 * time.Second)
	}

	pool.Stop()

	return report, nil
}
