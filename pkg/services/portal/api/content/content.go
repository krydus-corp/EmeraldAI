package content

import (
	"strings"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"

	common "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

func (c Content) Get(ctx echo.Context, userid, contentid, projectid string, datasetid *string, returnImage bool) (content *models.Content, imageBytes []byte, err error) {

	// Include annotation(s) associated with dataset
	if datasetid != nil {
		content, err = c.platform.ContentDB.ViewAnnotation(c.db, userid, contentid, *datasetid)
		if err != nil {
			return nil, nil, err
		}
	} else {
		content, err = c.platform.ContentDB.View(c.db, userid, contentid)
		if err != nil {
			return nil, nil, err
		}
	}

	// Check if content is associated with projectid
	if !(common.SliceContains(content.Projects, projectid)) {
		return nil, nil, platform.ErrContentDoesNotExist
	}

	if returnImage {
		contentBytes, err := c.blob.Get(content.StoredDir, content.StoredPath)
		if err != nil {
			return nil, nil, err
		}

		return content, contentBytes, nil
	}

	return content, nil, nil
}

func (c Content) ListAnnotated(ctx echo.Context, p models.Pagination, userid, projectid, datasetid, operator string, tagids ...string) ([]models.Content, int, error) {
	defaultOperator := "or"
	if strings.ToLower(operator) == "and" {
		defaultOperator = "and"
	}
	return c.platform.ContentDB.FindAnnotated(c.db, userid, projectid, datasetid, defaultOperator, p, tagids...)
}

func (c Content) Query(ctx echo.Context, userid string, q models.Query) ([]models.Content, int64, error) {
	return c.platform.ContentDB.Query(c.db, userid, q)
}

func (c Content) Sample(ctx echo.Context, userid, projectid string, count int, filterAnnotated bool) ([]models.Content, error) {
	content, err := c.platform.ContentDB.Sample(c.db, userid, projectid, count, filterAnnotated)
	if err != nil {
		return nil, err
	}

	return content, nil
}

// Delete deletes a content
func (c Content) Delete(ctc echo.Context, userid, projectid string, contentIds []string) error {
	// Check project exists
	_, err := c.platform.ProjectDB.View(c.db, userid, projectid)
	if err != nil {
		log.Errorf("unable to locate project for for user=%s, project=%s; not updating (err=%s)", userid, projectid, err.Error())
	}

	for _, contentId := range contentIds {
		// lookup content
		content, err := c.platform.ContentDB.View(c.db, userid, contentId)
		if err != nil {
			log.Errorf("unable to lookup content=%s for user=%s, project=%s; not removing (err=%s)", contentId, userid, projectid, err.Error())
			continue
		}

		// Check if project was associated with the content
		if !common.SliceContains(content.Projects, projectid) {
			log.Infof("content=%s not associated with project=%s", contentId, projectid)
			continue
		}

		content.Projects = common.RemoveFromSlice(content.Projects, projectid)
		if err = c.platform.ContentDB.Update(c.db, content); err != nil {
			log.Errorf("unable to remove project association for content=%s for user=%s, project=%s (err=%s)", contentId, userid, projectid, err.Error())
			continue
		}

		// Decrement Project count
		err = c.platform.ProjectDB.UpdateCount(c.db, projectid, userid, -1)
		if err != nil {
			log.Errorf("unable to update project for content=%s for user=%s, project=%s; not updating (err=%s)", contentId, userid, projectid, err.Error())
			continue
		}

		// Delete associated annotations - only if not locked!
		annotationIDs, err := c.platform.AnnotationDB.AnnotationsAssociatedWithContent(c.db, userid, projectid, contentId)
		if err != nil {
			log.Errorf("unable to remove annotation associated with content=%s for user=%s, project=%s; not updating (err=%s)", contentId, userid, projectid, err.Error())
			continue
		}

		if len(annotationIDs) > 0 {
			var objectIDs []primitive.ObjectID

			for _, idStr := range annotationIDs {
				id, err := primitive.ObjectIDFromHex(idStr)
				if err != nil {
					log.Errorf("unable to remove annotation associated with content=%s for user=%s, project=%s; not updating (err=%s)", contentId, userid, projectid, err.Error())
					continue
				}
				objectIDs = append(objectIDs, id)
			}

			if err := c.platform.AnnotationDB.DeleteUserAnnotations(c.db, userid, objectIDs); err != nil {
				log.Errorf("unable to remove annotation associated with content=%s for user=%s, project=%s; not updating (err=%s)", contentId, userid, projectid, err.Error())
				continue
			}
		}
	}

	return nil
}
