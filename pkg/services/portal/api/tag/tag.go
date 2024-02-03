// Package tag contains tag application services
package tag

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"

	common "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

var (
	isStringAlphabetic    = regexp.MustCompile(`^[a-zA-Z0-9_-]*$`).MatchString
	ErrTagNotAlphaNumeric = echo.NewHTTPError(http.StatusBadRequest, "Tag must be match the regex: `^[a-zA-Z0-9_-]*$`")
)

// Create creates a new tag entry
func (t Tag) Create(c echo.Context, req models.Tag) (models.Tag, error) {

	if !isStringAlphabetic(req.Name) {
		return models.Tag{}, ErrTagNotAlphaNumeric
	}

	// Check dataset exists
	if _, err := t.platform.DatasetDB.View(t.db, req.UserID, req.DatasetID); err != nil {
		return models.Tag{}, err
	}

	// Check project exists
	if _, err := t.platform.ProjectDB.View(t.db, req.UserID, req.ProjectID); err != nil {
		return models.Tag{}, err
	}

	// Database operation
	tag, err := t.platform.TagDB.Create(t.db, req)
	if err != nil {
		return models.Tag{}, err
	}

	return tag, nil
}

func (t Tag) List(c echo.Context, userid, datasetid string, p models.Pagination) ([]models.Tag, int64, error) {
	return t.platform.TagDB.List(t.db, userid, datasetid, p)
}

func (t Tag) Query(ctx echo.Context, userid string, q models.Query) ([]models.Tag, int64, error) {
	return t.platform.TagDB.Query(t.db, userid, q)
}

func (t Tag) View(c echo.Context, userid, tagid string) (models.Tag, error) {
	tag, err := t.platform.TagDB.View(t.db, userid, tagid)
	if err != nil {
		return models.Tag{}, err
	}

	return tag, nil
}

// Update contains tags's information used for updating
type Update struct {
	TagID    string
	UserID   string
	Name     string
	Property []string
}

// Update updates tag information
// !! When adding additional user updates, remember to propagate them to the DB layer !!
func (t Tag) Update(c echo.Context, r Update) (models.Tag, error) {
	id, err := primitive.ObjectIDFromHex(r.TagID)
	if err != nil {
		return models.Tag{}, err
	}

	properties := common.RemoveDuplicateStr(common.StringSliceToLower(r.Property))

	err = t.platform.TagDB.Update(t.db, models.Tag{
		ID:       id,
		UserID:   r.UserID,
		Name:     r.Name,
		Property: properties})
	if err != nil {
		return models.Tag{}, err
	}

	return t.platform.TagDB.View(t.db, r.UserID, r.TagID)
}

// Delete deletes the tag associated with a project and user
func (t Tag) Delete(c echo.Context, userid, tagid string) error {
	tag, err := t.platform.TagDB.View(t.db, userid, tagid)
	if err != nil {
		return err
	}

	dataset, err := t.platform.DatasetDB.View(t.db, userid, tag.DatasetID)
	if err != nil {
		return err
	}

	if dataset.Locked {
		return platform.ErrDatasetLocked
	}

	// Delete tag
	if err := t.platform.TagDB.Delete(t.db, tag.ID); err != nil {
		return err
	}

	// Delete tagid from annotations
	if err := t.platform.AnnotationDB.DeleteTagID(t.db, userid, tag.ProjectID, tag.DatasetID, tagid); err != nil {
		return err
	}

	return nil
}

func (t Tag) Properties(c echo.Context, userid, datasetid string) ([]string, error) {
	results, err := t.platform.TagDB.Distinct(t.db, userid, datasetid, "property")
	if err != nil {
		return nil, err
	}

	properties := []string{}
	for _, result := range results {
		if p, ok := result.(string); !ok {
			return nil, fmt.Errorf("unexpected type returned during property lookup; expected=string, returned=%s", reflect.TypeOf(result).String())
		} else {
			properties = append(properties, p)
		}
	}

	return properties, nil
}
