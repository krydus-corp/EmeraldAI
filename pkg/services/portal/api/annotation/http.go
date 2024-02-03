package annotation

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	errs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/error"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
)

// HTTP represents annotation http service
type HTTP struct {
	svc Service
}

// NewHTTP creates new annotation http service
// TODO: break out annotation update into a separate call
func NewHTTP(svc Service, r *echo.Group) {
	h := HTTP{svc}
	ur := r.Group("/annotations")

	// swagger:operation POST /v1/annotations annotations createAnnotationReq
	// ---
	// summary: Creates or updates an annotation.
	// description: |
	//  Creates or updates an annotation against the specified content for the specified tag(s).
	//  Annotations are unique to a project and dataset.
	// security:
	// - Bearer: []
	// consumes:
	//  - application/json
	// produces:
	//  - application/json
	// responses:
	//   "200":
	//     "schema":
	//      "$ref": "#/definitions/Annotation"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.POST("", h.create)

	// swagger:operation GET /v1/annotations annotations listAnnotationsReq
	// ---
	// summary: Returns list of annotations.
	// description: Returns list of annotations for the specified project and dataset.
	// security:
	// - Bearer: []
	// consumes:
	//  - application/json
	// produces:
	//  - application/json
	// parameters:
	// - name: project_id
	//   in: query
	//   type: string
	//   required: true
	// - name: dataset_id
	//   in: query
	//   type: string
	//   required: true
	// - name: limit
	//   in: query
	//   type: integer
	//   required: false
	// - name: page
	//   in: query
	//   type: integer
	//   required: false
	// - name: sort_key
	//   in: query
	//   type: string
	//   required: false
	// - name: sort_val
	//   in: query
	//   type: integer
	//   required: false
	// responses:
	//   "200":
	//     "$ref": "#/responses/listAnnotationsResp"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("", h.list)

	// swagger:operation GET /v1/annotations/{Id} annotations getAnnotationReq
	// ---
	// summary: Returns a single annotation.
	// description: Returns a single annotation by its ID.
	// security:
	// - Bearer: []
	// produces:
	//  - application/json
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of annotation
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "schema":
	//      "$ref": "#/definitions/Annotation"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("/:id", h.view)

	// swagger:operation DELETE /v1/annotations/{Id} annotations deleteAnnotationReq
	// ---
	// summary: Deletes an annotation.
	// description: Deletes an annotation with requested ID.
	// security:
	// - Bearer: []
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of annotation
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ok"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.DELETE("/:id", h.delete)

	// swagger:operation DELETE /v1/annotations annotations deleteAnnotationsReq
	// ---
	// summary: Deletes multiple annotations.
	// description: Deletes annotations with specified IDs.
	// security:
	// - Bearer: []
	// parameters:
	// - name: ids
	//   in: query
	//   type: array
	//   items:
	//     type: string
	//   required: true
	// schema:
	//  "$ref": "#/definitions/deleteAnnotationsReq"
	// responses:
	//   "200":
	//     "$ref": "#/responses/ok"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.DELETE("", h.deleteMany)

	// swagger:operation POST /v1/annotations/query annotations queryAnnotationsReq
	// ---
	// summary: Query annotations.
	// description: |
	//   Query annotations based on specified filter(s).
	//
	//   Parameters descriptions:
	//     - page: Indicates the current page; Indexed at 0
	//     - limit: Max number of items to return
	//     - sort_key: Field in model to sort on e.g. "created_at"
	//     - sort_val: Sort order; -1 for descending, 1 for ascending
	//     - operator: logical operator for defined filters. One of "or" || "and"
	//     - filters: Array of filters, where each filter is one of:
	//       - Simple Filter: Specify a `key` and `value` to query on e.g. `{"key": "_id", "value": "my-uuid"}`
	//       - Regex Filter: In addition to `key` and `value`, specify a filter of type `regex` with the `enable` flag set to `true` e.g. `"regex": {"enable": true, "options": "i"}`
	//         Available options include:
	//           - `i` (case insensitivity),
	//           - `m` (patterns that include anchors),
	//           - `x` (ignore all white space characters in the pattern unless escaped)
	//           - `s` (allows the dot character (i.e. .) to match all characters including newline characters.)
	//       - Datetime Filter: In addition to `key` and `value`, specify a filter of type `datetime` with the `enable` flag set to `true` e.g. `"datetime": {"enable": true, "options": "$gt"}`
	//         Available options include:
	//           - `$gt` (greater than),
	//           - `$gte` (greater than or equal to),
	//           - `$lt` (less than)
	//           - `$lte` (less than or equal to)
	//
	//   Regex Example:
	//   ```json
	//   {
	//       "limit": 10,
	//       "page": 0,
	//       "sort_key": "created_at",
	//       "sort_val": -1,
	//       "operator": "and",
	//       "filters": [
	//           {
	//               "key": "name",
	//               "regex": {
	//                   "enable": true,
	//                   "options": "i"
	//               },
	//               "value": ".*animal.*"
	//           },
	//           {
	//               "key": "description",
	//               "value": ".*cat.*"
	//               "regex": {
	//                 "enable": true,
	//                   "options": "i"
	//           },
	//       ]
	//   }
	//   ```
	//
	//   Datetime Example:
	//   ```json
	//   {
	//       "limit": 10,
	//       "page": 0,
	//       "sort_key": "created_at",
	//       "sort_val": -1,
	//       "operator": "and",
	//       "filters": [
	//           {
	//               "key": "created_at",
	//               "value": "2022-07-06T17:30:07+00:00",
	//               "datetime": {
	//                   "enable": true,
	//                   "options": "$lt"
	//               }
	//           },
	//           {
	//               "key": "created_at",
	//               "value": "2022-07-01T02:24:23.204Z",
	//               "datetime": {
	//                   "enable": true,
	//                   "options": "$gte"
	//               }
	//           }
	//      ]
	//   }
	//   ```
	// security:
	// - Bearer: []
	// consumes:
	//  - application/json
	// produces:
	//  - application/json
	// schema:
	//  "$ref": "#/definitions/queryAnnotationsReq"
	// responses:
	//   "200":
	//     "$ref": "#/responses/queryAnnotationsResp"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.POST("/query", h.query)

	// swagger:operation GET /v1/annotations/statistics annotations getAnnotationStatisticsReq
	// ---
	// summary: Get annotation statistics.
	// description: Get annotation statistics for a given dataset & project.
	// security:
	// - Bearer: []
	// produces:
	//  - application/json
	// schema:
	//  "$ref": "#/definitions/getAnnotationStatisticsReq"
	// responses:
	//   "200":
	//     "$ref": "#/responses/annotationStatisticsResp"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("/statistics", h.stats)
}

// Annotation create request
// swagger:parameters createAnnotationReq
type CreateAnnotationReq struct {
	// in:body
	Body struct {
		//	ID of content
		ContentID string `json:"content_id" query:"content_id" validate:"required"`
		//	Tag ID
		TagID []string `json:"tag_id" query:"tag_id"`
		//	Project ID
		ProjectID string `json:"project_id" query:"project_id" validate:"required"`
		//	Dataset ID
		DatasetID string `json:"dataset_id" query:"dataset_id" validate:"required"`
		// Metadata
		Metadata models.AnnotationMetadata `json:"metadata" query:"metadata"`
		// ThumbnailSize
		ThumbnailSize int `json:"thumbnail_size,omitempty" validate:"oneof=0 100 200 640"`
	}
}

func (h HTTP) create(c echo.Context) error {
	r := new(CreateAnnotationReq).Body
	if err := c.Bind(&r); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)

	annotation, err := h.svc.Create(c, models.Annotation{
		UserID:    user.ID.Hex(),
		ProjectID: r.ProjectID,
		DatasetID: r.DatasetID,
		ContentID: r.ContentID,
		TagIDs:    r.TagID,
		Metadata:  r.Metadata,
	}, r.ThumbnailSize)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, annotation)
}

// Annotations list response
// swagger:response listAnnotationsResp
type listAnnotationsResp struct {
	// in:body
	Body struct {
		Annotations []models.Annotation `json:"annotations"`
		Page        int                 `json:"page"`
		Count       int64               `json:"count"`
	}
}

type listAnnotationsReq struct {
	models.PaginationReq
	ProjectID string `json:"project_id" query:"project_id" validate:"required"`
	DatasetID string `json:"dataset_id" query:"dataset_id" validate:"required"`
}

func (h HTTP) list(c echo.Context) error {
	var req listAnnotationsReq

	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)

	result, count, err := h.svc.List(c, user.ID.Hex(), req.ProjectID, req.DatasetID, req.Transform())
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	resp := listAnnotationsResp{struct {
		Annotations []models.Annotation "json:\"annotations\""
		Page        int                 "json:\"page\""
		Count       int64               "json:\"count\""
	}{result, req.Page, count}}

	return c.JSON(http.StatusOK, resp.Body)
}

func (h HTTP) view(c echo.Context) error {
	id := c.Param("id")
	user := c.Get("current_user").(models.User)

	result, err := h.svc.View(c, user.ID.Hex(), id)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, result)
}

func (h HTTP) delete(c echo.Context) error {
	id := c.Param("id")
	user := c.Get("current_user").(models.User)

	if err := h.svc.Delete(c, user.ID.Hex(), id); err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": fmt.Sprintf("Annotation %s deleted", id)})
}

type deleteAnnotationsReq struct {
	IDs []string `json:"ids" query:"ids" validate:"gt=0,unique,dive,hexadecimal,required"`
}

func (h HTTP) deleteMany(c echo.Context) error {
	user := c.Get("current_user").(models.User)

	var req deleteAnnotationsReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	if err := h.svc.Delete(c, user.ID.Hex(), req.IDs...); err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": "Annotations deleted"})
}

// Annotations query request
// swagger:parameters queryAnnotationsReq
//
//lint:ignore U1000 ignore, used for swagger spec
type queryAnnotationsReq struct {
	// in:body
	Body struct {
		models.QueryReq `json:"query"`
		Page            int `json:"page"`
	}
}

// Annotations query response
// swagger:response queryAnnotationsResp
type queryAnnotationsResp struct {
	// in: body
	Body struct {
		Annotations []models.Annotation `json:"annotations"`
		Page        int                 `json:"page"`
		Count       int64               `json:"count"`
	}
}

func (h HTTP) query(c echo.Context) error {
	var req models.QueryReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)

	annotations, count, err := h.svc.Query(c, user.ID.Hex(), req.Transform())
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	resp := queryAnnotationsResp{struct {
		Annotations []models.Annotation "json:\"annotations\""
		Page        int                 "json:\"page\""
		Count       int64               "json:\"count\""
	}{annotations, req.Page, count}}

	return c.JSON(http.StatusOK, resp.Body)
}

// Annotations statistics request
// swagger:parameters getAnnotationStatisticsReq
type getAnnotationStatisticsReq struct {
	// Project id
	//
	// in: query
	// required: true
	ProjectID string `json:"project_id" query:"project_id" validate:"required"`
	// Dataset id
	//
	// in: query
	// required: true
	DatasetID string `json:"dataset_id" query:"dataset_id" validate:"required"`
	// Comma delimited list of statistics to return. When ommited, all statistics will be return.
	//
	// in: query
	// required: false
	StatsToReturn *string `json:"stats" query:"stats"`
}

// swagger:response
type annotationStatisticsResp struct {
	// in: body
	Body struct {
		Statistics `json:"statistics"`
	}
}

func (h HTTP) stats(c echo.Context) error {
	var req getAnnotationStatisticsReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)

	stats, err := h.svc.Statistics(c, user.ID.Hex(), req.ProjectID, req.DatasetID, req.StatsToReturn)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	resp := annotationStatisticsResp{struct {
		Statistics "json:\"statistics\""
	}{*stats}}

	return c.JSON(http.StatusOK, resp.Body)
}
