// Package annotation contains annotation application services
package annotation

import (
	"reflect"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/sync/errgroup"

	common "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	thumbnail "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/image"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

const (
	jumbo  = 1048576 // 1024 x 1024
	large  = 518400  // 720 x 720
	medium = 65536   // 256 x 256
	small  = 10000   // 100 x 100
)

// Create creates a new annotation
func (a Annotation) Create(c echo.Context, req models.Annotation, thumbnailSize int) (*models.Annotation, error) {
	var annotation *models.Annotation

	// Check project exists
	project, err := a.platform.ProjectDB.View(a.db, req.UserID, req.ProjectID)
	if err != nil {
		return nil, err
	}

	// Check dataset exists
	if dataset, err := a.platform.DatasetDB.View(a.db, req.UserID, req.DatasetID); err != nil {
		return nil, err
	} else if dataset.Locked {
		return nil, echo.NewHTTPError(409, "Dataset Locked")
	}

	// Check content exists
	content, err := a.platform.ContentDB.View(a.db, req.UserID, req.ContentID)
	if err != nil {
		return nil, err
	}

	// Check if content is associated with projectid
	if !(common.SliceContains(content.Projects, req.ProjectID)) {
		return nil, platform.ErrContentDoesNotExist
	}

	// Check tags exists
	for _, tag := range req.TagIDs {
		if _, err := a.platform.TagDB.View(a.db, req.UserID, tag); err != nil {
			return nil, err
		}
	}

	// Check annotation exists
	annotation, err = a.platform.AnnotationDB.FindContentAnnotation(
		a.db,
		req.UserID,
		req.ProjectID,
		req.DatasetID,
		req.ContentID,
	)
	if err != nil && err != platform.ErrAnnotationDoesNotExist {
		return nil, err
	}

	// Add or update annotations
	if annotation != nil {

		// Update annotation fields
		annotation.TagIDs = req.TagIDs
		annotation.Metadata = req.Metadata

		// Validate
		if err := annotation.Valid(project.AnnotationType); err != nil {
			return nil, err
		}

		// Update Base64 img.
		contentBytes, err := a.blob.Get(content.StoredDir, content.StoredPath)
		if err != nil {
			return nil, err
		}
		boundingBoxes := []thumbnail.BoundingBox{}
		tagmap := make(map[string]string)
		for _, box := range req.Metadata.BoundingBoxes {
			// Check for previous tag names before lookup.
			if tagName, ok := tagmap[box.TagID]; ok {
				boundingBoxes = append(boundingBoxes, thumbnail.BoundingBox{
					Xmin:      box.Xmin,
					Xmax:      box.Xmax,
					Ymin:      box.Ymin,
					Ymax:      box.Ymax,
					ClassName: tagName,
				})
			} else {
				tag, err := a.platform.TagDB.View(a.db, content.UserID, box.TagID)
				if err != nil {
					return nil, err
				}
				tagmap[tag.ID.Hex()] = tag.Name
				boundingBoxes = append(boundingBoxes, thumbnail.BoundingBox{
					Xmin:      box.Xmin,
					Xmax:      box.Xmax,
					Ymin:      box.Ymin,
					Ymax:      box.Ymax,
					ClassName: tag.Name,
				})
			}
		}
		imgBase64, err := createBase64IMG(contentBytes, boundingBoxes, project.AnnotationType, thumbnailSize)
		if err != nil {
			return nil, err
		}
		annotation.Base64Image = imgBase64

		log.Infof("Annotation b64=%s", annotation.Base64Image)
		log.Infof("Annotation meta=%v", annotation.Metadata)

		// Update annotation at DB
		err = a.platform.AnnotationDB.Update(a.db, annotation)
		if err != nil {
			return nil, errors.Wrapf(err, "failed annotating content=%s, tag=%s, user=%s", req.ContentID, req.TagIDs, req.UserID)
		}
	} else {
		// Update Base64 img.
		contentBytes, err := a.blob.Get(content.StoredDir, content.StoredPath)
		if err != nil {
			return nil, err
		}
		boundingBoxes := []thumbnail.BoundingBox{}
		tagmap := make(map[string]string)
		for _, box := range req.Metadata.BoundingBoxes {
			// Check for previous tag names before lookup.
			if tagName, ok := tagmap[box.TagID]; ok {
				boundingBoxes = append(boundingBoxes, thumbnail.BoundingBox{
					Xmin:      box.Xmin,
					Xmax:      box.Xmax,
					Ymin:      box.Ymin,
					Ymax:      box.Ymax,
					ClassName: tagName,
				})
			} else {
				tag, err := a.platform.TagDB.View(a.db, content.UserID, box.TagID)
				if err != nil {
					return nil, err
				}
				tagmap[tag.ID.Hex()] = tag.Name
				boundingBoxes = append(boundingBoxes, thumbnail.BoundingBox{
					Xmin:      box.Xmin,
					Xmax:      box.Xmax,
					Ymin:      box.Ymin,
					Ymax:      box.Ymax,
					ClassName: tag.Name,
				})
			}
		}
		imgBase64, err := createBase64IMG(contentBytes, boundingBoxes, project.AnnotationType, thumbnailSize)
		if err != nil {
			return nil, err
		}
		// Add annotation
		annotation = models.NewAnnotation(
			req.UserID,
			req.ProjectID,
			req.DatasetID,
			req.ContentID,
			req.TagIDs,
			imgBase64,
			req.Metadata,
			models.ContentMetadata{Size: content.Size, Height: content.Height, Width: content.Width},
		)

		// Validate
		if err := annotation.Valid(project.AnnotationType); err != nil {
			return nil, err
		}

		// Create at DB
		if _, err := a.platform.AnnotationDB.Create(a.db, *annotation); err != nil {
			return nil, errors.Wrapf(err, "failed annotating content=%s, tag=%s, user=%s", req.ContentID, req.TagIDs, req.UserID)
		}
	}

	return a.platform.AnnotationDB.View(a.db, req.UserID, annotation.ID.Hex())
}

func (a Annotation) Query(ctx echo.Context, userid string, q models.Query) ([]models.Annotation, int64, error) {
	return a.platform.AnnotationDB.Query(a.db, userid, q)
}

// List returns list of annotations
func (a Annotation) List(c echo.Context, userid, projectid, datasetid string, p models.Pagination) ([]models.Annotation, int64, error) {
	// Check project exists
	_, err := a.platform.ProjectDB.View(a.db, userid, projectid)
	if err != nil {
		return []models.Annotation{}, 0, err
	}
	// Check dataset exists
	_, err = a.platform.DatasetDB.View(a.db, userid, datasetid)
	if err != nil {
		return []models.Annotation{}, 0, err
	}

	return a.platform.AnnotationDB.List(a.db, userid, projectid, datasetid, p)
}

// View returns single annotation
func (a Annotation) View(c echo.Context, userid, id string) (*models.Annotation, error) {
	return a.platform.AnnotationDB.View(a.db, userid, id)
}

// Delete deletes a annotation
func (a Annotation) Delete(c echo.Context, userid string, annotationids ...string) error {

	var objectIDs []primitive.ObjectID

	for _, idStr := range annotationids {
		id, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			return platform.ErrAnnotationDoesNotExist
		}
		objectIDs = append(objectIDs, id)
	}

	// Delete annotations
	if err := a.platform.AnnotationDB.DeleteUserAnnotations(a.db, userid, objectIDs); err != nil {
		return err
	}

	return nil
}

// Statistics returns all annotation level statistics
func (a Annotation) Statistics(c echo.Context, userid, projectid, datasetid string, statsToReturn *string) (*Statistics, error) {
	// Check project exists
	_, err := a.platform.ProjectDB.View(a.db, userid, projectid)
	if err != nil {
		return nil, err
	}
	// Check dataset exists
	_, err = a.platform.DatasetDB.View(a.db, userid, datasetid)
	if err != nil {
		return nil, err
	}

	// Parse stats to return
	statsSlice := []string{}
	if statsToReturn != nil {
		tmp := strings.Split(*statsToReturn, ",")
		for _, stat := range tmp {
			statsSlice = append(statsSlice, strings.ToLower(strings.TrimSpace(stat)))
		}
	}

	stats := Statistics{}

	// Parallelize stat requests
	// Note: Synchronization not required as, while we are performing write access,
	// they are all on different fields within `stats` and different struct fields have different memory locations.
	var g errgroup.Group

	// Total annotated images

	if len(statsSlice) == 0 || getStat("TotalAnnotatedImages", statsSlice, &Statistics{}) {
		g.Go(func() error {
			var err error
			if stats.TotalAnnotatedImages, err = a.platform.AnnotationDB.CountAnnotations(a.db, userid, datasetid); err != nil {
				return err
			}
			return nil
		})
	}
	// Total unannotated images
	if len(statsSlice) == 0 || getStat("TotalUnannotatedImages", statsSlice, &Statistics{}) {
		g.Go(func() error {
			var err error
			if stats.TotalUnannotatedImages, err = a.platform.AnnotationDB.CountUnannotated(a.db, userid, datasetid, projectid); err != nil {
				return err
			}
			return nil
		})
	}

	// Total annotations
	if len(statsSlice) == 0 || getStat("TotalAnnotations", statsSlice, &Statistics{}) {
		g.Go(func() error {
			var err error
			if stats.TotalAnnotations, err = a.totalAnnotations(userid, projectid, datasetid); err != nil {
				return err
			}
			return nil
		})
	}
	// Average annotations per image
	if len(statsSlice) == 0 || getStat("AverageAnnotationsPerImage", statsSlice, &Statistics{}) {
		g.Go(func() error {
			var err error
			if stats.AverageAnnotationsPerImage, err = a.averageAnnotationsPerImage(userid, projectid, datasetid); err != nil {
				return err
			}
			return nil
		})
	}
	// Total annotated images with no tag associations
	if len(statsSlice) == 0 || getStat("AnnotatedNullTags", statsSlice, &Statistics{}) {
		g.Go(func() error {
			var err error
			if stats.AnnotatedNullTags, err = a.nullAnnotations(userid, projectid, datasetid); err != nil {
				return err
			}
			return nil
		})
	}
	// Annotations per class
	if len(statsSlice) == 0 || getStat("AnnotationsPerClass", statsSlice, &Statistics{}) {
		g.Go(func() error {
			var err error
			if stats.AnnotationsPerClass, err = a.annotationsPerClass(userid, projectid, datasetid); err != nil {
				return err
			}
			return nil
		})
	}
	// Average image height
	if len(statsSlice) == 0 || getStat("AverageImageHeightPixels", statsSlice, &Statistics{}) {
		g.Go(func() error {
			var err error
			if stats.AverageImageHeightPixels, err = a.imageStat(userid, datasetid, "avg", "height"); err != nil {
				return err
			}
			return nil
		})
	}
	// Average image width
	if len(statsSlice) == 0 || getStat("AverageImageWidthPixels", statsSlice, &Statistics{}) {
		g.Go(func() error {
			var err error
			if stats.AverageImageWidthPixels, err = a.imageStat(userid, datasetid, "avg", "width"); err != nil {
				return err
			}
			return nil
		})
	}
	// Min image height
	if len(statsSlice) == 0 || getStat("MinImageHeightPixels", statsSlice, &Statistics{}) {
		g.Go(func() error {
			var err error
			if stats.MinImageHeightPixels, err = a.imageStat(userid, datasetid, "min", "height"); err != nil {
				return err
			}
			return nil
		})
	}
	// Min image width
	if len(statsSlice) == 0 || getStat("MinImageWidthPixels", statsSlice, &Statistics{}) {
		g.Go(func() error {
			var err error
			if stats.MinImageWidthPixels, err = a.imageStat(userid, datasetid, "min", "width"); err != nil {
				return err
			}
			return nil
		})
	}
	// Max image height
	if len(statsSlice) == 0 || getStat("MaxImageHeightPixels", statsSlice, &Statistics{}) {
		g.Go(func() error {
			var err error
			if stats.MaxImageHeightPixels, err = a.imageStat(userid, datasetid, "max", "height"); err != nil {
				return err
			}
			return nil
		})
	}
	// Max image width
	if len(statsSlice) == 0 || getStat("MaxImageWidthPixels", statsSlice, &Statistics{}) {
		g.Go(func() error {
			var err error
			if stats.MaxImageWidthPixels, err = a.imageStat(userid, datasetid, "max", "width"); err != nil {
				return err
			}
			return nil
		})
	}
	// AverageImageSizeBytes
	if len(statsSlice) == 0 || getStat("AverageImageSizeBytes", statsSlice, &Statistics{}) {
		g.Go(func() error {
			var err error
			if stats.AverageImageSizeBytes, err = a.imageStat(userid, datasetid, "avg", "size"); err != nil {
				return err
			}
			return nil
		})
	}
	// MinImageSizeBytes
	if len(statsSlice) == 0 || getStat("MinImageSizeBytes", statsSlice, &Statistics{}) {
		g.Go(func() error {
			var err error
			if stats.MinImageSizeBytes, err = a.imageStat(userid, datasetid, "min", "size"); err != nil {
				return err
			}
			return nil
		})
	}
	// MaxImageSizeBytes
	if len(statsSlice) == 0 || getStat("MaxImageSizeBytes", statsSlice, &Statistics{}) {
		g.Go(func() error {
			var err error
			if stats.MaxImageSizeBytes, err = a.imageStat(userid, datasetid, "max", "size"); err != nil {
				return err
			}
			return nil
		})
	}
	// Image insights
	if len(statsSlice) == 0 || getStat("DimensionInsights", statsSlice, &Statistics{}) {
		g.Go(func() error {
			var err error
			if stats.DimensionInsights, err = a.imageInsights(userid, projectid, datasetid); err != nil {
				return err
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &stats, nil
}

func (a Annotation) totalAnnotations(userid, projectid, datasetid string) (*int64, error) {
	var results []struct {
		Count     int64 `bson:"count"`
		NullCount int64 `bson:"null_count"`
	}
	if err := a.platform.AnnotationDB.TotalAnnotations(a.db, userid, projectid, datasetid, &results); err != nil {
		log.Errorf("total annotations err=%s", err.Error())
		return nil, err
	}

	if len(results) > 0 {
		return common.Ptr(results[0].Count + results[0].NullCount), nil
	}
	return common.Ptr(int64(0)), nil
}

func (a Annotation) averageAnnotationsPerImage(userid, projectid, datasetid string) (*float64, error) {
	var results []struct {
		AverageAnnotations float64 `bson:"average_annotations"`
	}

	if err := a.platform.AnnotationDB.AverageAnnotationsPerImage(a.db, userid, projectid, datasetid, &results); err != nil {
		log.Errorf("annotations per image err=%s", err.Error())
		return nil, err
	}

	if len(results) > 0 {
		return common.Ptr(results[0].AverageAnnotations), nil
	}

	return common.Ptr(0.0), nil
}

func (a Annotation) nullAnnotations(userid, projectid, datasetid string) (*int64, error) {
	return a.platform.AnnotationDB.CountNullAnnotations(a.db, userid, projectid, datasetid)
}

func (a Annotation) annotationsPerClass(userid, projectid, datasetid string) (*map[string]annotationClassData, error) {
	var results []struct {
		ID    string `bson:"_id"`
		Count int    `bson:"count"`
	}
	if err := a.platform.AnnotationDB.AnnotationsPerClass(a.db, userid, projectid, datasetid, &results); err != nil {
		log.Errorf("annotation per class err=%s", err.Error())
		return nil, err
	}
	var resultsFmt = map[string]annotationClassData{}
	tagIDs := []string{}
	tagCounts := []int{}
	for _, r := range results {
		tag, err := a.platform.TagDB.View(a.db, userid, r.ID)
		if err != nil {
			return nil, err
		}

		annotationData := annotationClassData{Name: tag.Name, Count: r.Count}
		tagIDs = append(tagIDs, r.ID)
		tagCounts = append(tagCounts, r.Count)
		resultsFmt[r.ID] = annotationData
	}

	nullAnnotations, err := a.nullAnnotations(userid, projectid, datasetid)
	if err != nil {
		return nil, err
	}

	nullAnnotationClassData := annotationClassData{Name: "null", Count: int(*nullAnnotations)}
	tagIDs = append(tagIDs, "null")
	tagCounts = append(tagCounts, int(*nullAnnotations))
	resultsFmt["null"] = nullAnnotationClassData
	balances := common.FindBalances(tagCounts)
	if len(balances) > 0 {
		for i, r := range tagIDs {
			annotationData := resultsFmt[r]
			annotationData.Balance = balances[i]
			resultsFmt[r] = annotationData
		}
	}
	return &resultsFmt, nil
}

func (a Annotation) imageStat(userid, datasetid, stat, field string) (*int64, error) {
	var results []struct {
		Stat float64 `bson:"stat"`
	}

	if err := a.platform.AnnotationDB.AnnotationsImageStat(a.db, userid, datasetid, stat, field, &results); err != nil {
		log.Errorf("annotations image stat err=%s", err.Error())
		return nil, err
	}

	if len(results) > 0 {
		return common.Ptr(int64(results[0].Stat)), nil
	}

	return common.Ptr(int64(0)), nil
}

func (a Annotation) imageInsights(userid, projectid, datasetid string) (*annotationDimensionInsights, error) {
	var dimensions []annotationDimensions

	if err := a.platform.AnnotationDB.AnnotationsImageInsights(a.db, userid, projectid, datasetid, &dimensions); err != nil {
		log.Errorf("image insights error=%s", err.Error())
		return nil, err
	}

	insights := annotationDimensionInsights{
		Dimensions: dimensions,
	}

	// Ratio and size
	var (
		widths  []int
		heights []int
	)

	for _, dimension := range dimensions {
		widths = append(widths, dimension.Width)
		heights = append(heights, dimension.Height)

		// Aspect Ratio
		if dimension.Height > dimension.Width {
			insights.AspectRatioDistributions.Tall.Count += 1
		} else if dimension.Width > dimension.Height {
			insights.AspectRatioDistributions.Wide.Count += 1
		} else {
			insights.AspectRatioDistributions.Square.Count += 1
		}

		// Size distribution
		area := dimension.Height * dimension.Width
		switch {
		case area >= jumbo:
			insights.SizeDistributions.Jumbo.Count += 1
		case area >= large:
			insights.SizeDistributions.Large.Count += 1
		case area >= medium:
			insights.SizeDistributions.Medium.Count += 1
		case area >= small:
			insights.SizeDistributions.Small.Count += 1
		default:
			insights.SizeDistributions.Tiny.Count += 1
		}
	}

	// Aspect Ratio zscores
	aspectRatioCounts := []int{
		insights.AspectRatioDistributions.Tall.Count,
		insights.AspectRatioDistributions.Wide.Count,
		insights.AspectRatioDistributions.Square.Count,
	}
	zscoreAspectRatioCounts := []int{}
	zscoreAspectRatioTags := []string{}
	for i, aspectRatioCount := range aspectRatioCounts {
		switch i {
		case 0:
			zscoreAspectRatioCounts = append(zscoreAspectRatioCounts, aspectRatioCount)
			zscoreAspectRatioTags = append(zscoreAspectRatioTags, "tall")
		case 1:
			zscoreAspectRatioCounts = append(zscoreAspectRatioCounts, aspectRatioCount)
			zscoreAspectRatioTags = append(zscoreAspectRatioTags, "wide")
		case 2:
			zscoreAspectRatioCounts = append(zscoreAspectRatioCounts, aspectRatioCount)
			zscoreAspectRatioTags = append(zscoreAspectRatioTags, "square")

		}
	}
	balances := common.FindBalances(zscoreAspectRatioCounts)
	if len(balances) > 0 {
		for i, balance := range balances {
			switch zscoreAspectRatioTags[i] {
			case "tall":
				sizeData := insights.AspectRatioDistributions.Tall
				sizeData.Balance = balance
				insights.AspectRatioDistributions.Tall = sizeData
			case "wide":
				sizeData := insights.AspectRatioDistributions.Wide
				sizeData.Balance = balance
				insights.AspectRatioDistributions.Wide = sizeData
			case "square":
				sizeData := insights.AspectRatioDistributions.Square
				sizeData.Balance = balance
				insights.AspectRatioDistributions.Square = sizeData
			}
		}
	}

	// Size distribution zscores
	sizeDistributionCounts := []int{
		insights.SizeDistributions.Tiny.Count,
		insights.SizeDistributions.Small.Count,
		insights.SizeDistributions.Medium.Count,
		insights.SizeDistributions.Large.Count,
		insights.SizeDistributions.Jumbo.Count,
	}
	zscoreSizeDistributionsCounts := []int{}
	zscoreSizeDistributionTags := []string{}
	for i, sizeDistributionCount := range sizeDistributionCounts {
		switch i {
		case 0:
			zscoreSizeDistributionsCounts = append(zscoreSizeDistributionsCounts, sizeDistributionCount)
			zscoreSizeDistributionTags = append(zscoreSizeDistributionTags, "tiny")
		case 1:
			zscoreSizeDistributionsCounts = append(zscoreSizeDistributionsCounts, sizeDistributionCount)
			zscoreSizeDistributionTags = append(zscoreSizeDistributionTags, "small")
		case 2:
			zscoreSizeDistributionsCounts = append(zscoreSizeDistributionsCounts, sizeDistributionCount)
			zscoreSizeDistributionTags = append(zscoreSizeDistributionTags, "medium")
		case 3:
			zscoreSizeDistributionsCounts = append(zscoreSizeDistributionsCounts, sizeDistributionCount)
			zscoreSizeDistributionTags = append(zscoreSizeDistributionTags, "large")

		case 4:
			zscoreSizeDistributionsCounts = append(zscoreSizeDistributionsCounts, sizeDistributionCount)
			zscoreSizeDistributionTags = append(zscoreSizeDistributionTags, "jumbo")
		}
	}
	balances = common.FindBalances(zscoreSizeDistributionsCounts)
	if len(balances) > 0 {
		for i, balance := range balances {
			switch zscoreSizeDistributionTags[i] {
			case "tiny":
				sizeData := insights.SizeDistributions.Tiny
				sizeData.Balance = balance
				insights.SizeDistributions.Tiny = sizeData
			case "small":
				sizeData := insights.SizeDistributions.Small
				sizeData.Balance = balance
				insights.SizeDistributions.Small = sizeData
			case "medium":
				sizeData := insights.SizeDistributions.Medium
				sizeData.Balance = balance
				insights.SizeDistributions.Medium = sizeData
			case "large":
				sizeData := insights.SizeDistributions.Large
				sizeData.Balance = balance
				insights.SizeDistributions.Large = sizeData
			case "jumbo":
				sizeData := insights.SizeDistributions.Jumbo
				sizeData.Balance = balance
				insights.SizeDistributions.Jumbo = sizeData
			}
		}
	}

	widths_length := len(widths)
	heights_length := len(heights)

	if widths_length == 0 || heights_length == 0 {
		return &insights, nil
	}

	medians := common.Median(widths, heights)
	insights.MedianWidth, insights.MedianHeight = float64(medians[0]), float64(medians[1])

	return &insights, nil
}

func getStat(fieldName string, tags []string, i any) bool {
	field, ok := reflect.TypeOf(i).Elem().FieldByName(fieldName)
	if !ok {
		// Intentional panic - misuse of function if passing in fieldname that does not exist in `i`
		panic("Field not found")
	}

	jsonTag := common.GetStructTag(field, "json")
	jsonTag = strings.Split(jsonTag, ",")[0]

	return common.SliceContains(tags, jsonTag)
}

func createBase64IMG(contentBytes []byte, boundingBoxes []thumbnail.BoundingBox, projectType string, thumbnailSize int) (string, error) {
	if thumbnailSize == 0 {
		thumbnailSize = thumbnail.BoundingBoxDefaultThumbnailSize
	}
	if projectType == models.ProjectAnnotationTypeClassification.String() {
		thumb, _, err := thumbnail.Thumbnail(contentBytes, 100, 100)
		if err != nil {
			return "", err
		}
		return thumb, nil
	} else if projectType == models.ProjectAnnotationTypeBoundingBox.String() {
		thumb, _, err := thumbnail.ThumbnailBoundingBox(
			contentBytes,
			thumbnail.BoundingBoxDefaultThumbnailSize,
			thumbnail.BoundingBoxDefaultThumbnailSize,
			boundingBoxes,
		)
		if err != nil {
			return "", err
		}
		return thumb, nil
	}

	return "", nil
}
