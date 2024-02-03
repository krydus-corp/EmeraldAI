package content

import (
	"bytes"
	"io"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	errs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/error"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
)

// HTTP represents user http service
type HTTP struct {
	svc Service
}

func NewHTTP(svc Service, r *echo.Group) {
	h := HTTP{svc}
	ur := r.Group("/content")

	// swagger:operation GET /v1/content/{ContentID} content viewContentReq
	// ---
	// summary: Returns a single content.
	// description: Returns a single content by it's ID.
	// security:
	// - Bearer: []
	// produces:
	//  - application/json
	// schema:
	//  "$ref": "#/definitions/viewContentReq"
	// responses:
	//   "200":
	//     "schema":
	//      "$ref": "#/definitions/Content"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("/:id", h.view)

	// swagger:operation GET /v1/content/annotated content listAnnotatedContentReq
	// ---
	// summary: Returns list of content that has been annotated.
	// description: |
	//   Returns list of content that has been annotated i.e. associated with a dataset. The annotation will also be included, embedded in the Content model.
	//   If the optional dataset_id parameter is omitted, this will only return content not associated with a dataset i.e. content that has not been annotated.
	// security:
	// - Bearer: []
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: project_id
	//   in: query
	//   type: string
	//   required: true
	// - name: dataset_id
	//   in: query
	//   type: string
	//   required: false
	// - name: tag_id
	//   in: query
	//   type: array
	//   items:
	//     type: string
	//   required: false
	// - name: operator
	//   description: Logical operator for tag_id parameter. Can be one of "and" or "or".
	//   in: query
	//   type: string
	//   default: or
	//   required: false
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
	//     "$ref": "#/responses/listAnnotatedContentResp"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("/annotated", h.listAnnotated)

	// swagger:operation GET /v1/content/sample content sampleContentReq
	// ---
	// summary: Returns random sample of content.
	// description: Returns a random list of content up to the specified count. The return content is not attached to a dataste i.e. is not annotated yet.
	// security:
	// - Bearer: []
	// produces:
	//  - application/json
	// schema:
	//  "$ref": "#/definitions/sampleContentReq"
	// responses:
	//   "200":
	//     "$ref": "#/responses/sampleContentResp"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("/sample", h.sample)

	// swagger:operation POST /v1/content/query content queryContentReq
	// ---
	// summary: Query content.
	// description: |
	//   Query content based on specified filter(s).
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
	//  "$ref": "#/definitions/queryContentReq"
	// responses:
	//   "200":
	//     "$ref": "#/responses/queryContentResp"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.POST("/query", h.query)

	// swagger:operation DELETE /v1/content/delete content deleteContentReq
	// ---
	// summary: Deletes a content.
	// description: Deletes a content with requested ID.
	// security:
	// - Bearer: []
	// schema:
	//  "$ref": "#/definitions/deleteContentReq"
	// responses:
	//   "200":
	//     "$ref": "#/responses/ok"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.DELETE("/delete", h.delete)
}

// Content list response
// swagger:response listAnnotatedContentResp
type listAnnotatedContentResp struct {
	// in:body
	Body struct {
		Content []models.Content `json:"content"`
		Count   int              `json:"count"`
		Page    int              `json:"page"`
	}
}

type listAnnotatedContentReq struct {
	models.PaginationReq
	ProjectID string   `json:"project_id" query:"project_id" validate:"required"`
	DatasetID string   `json:"dataset_id" query:"dataset_id"`
	TagID     []string `json:"tag_id" query:"tag_id"`
	Operator  string   `json:"operator" query:"operator"`
}

func (h HTTP) listAnnotated(c echo.Context) error {
	var req listAnnotatedContentReq

	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)

	result, count, err := h.svc.ListAnnotated(c, req.Transform(), user.ID.Hex(), req.ProjectID, req.DatasetID, req.Operator, req.TagID...)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	resp := listAnnotatedContentResp{struct {
		Content []models.Content "json:\"content\""
		Count   int              "json:\"count\""
		Page    int              "json:\"page\""
	}{result, count, req.Page}}

	return c.JSON(http.StatusOK, resp.Body)
}

// Content view request
// swagger:parameters viewContentReq
type viewContentReq struct {
	//	ID of content
	//  in: path
	//  type: string
	//  required: true
	ContentID string `param:"id" validate:"required"`
	//	ID of project
	//  in: query
	//  type: string
	//  required: true
	ProjectID string `query:"project_id" json:"project_id" validate:"required"`
	//	ID of Dataset. If specified, will return any annotation(s) associated with the content and dataset
	//  in: query
	//  type: string
	//  required: false
	DatasetID *string `query:"dataset_id" json:"dataset_id"`
	//	Include Image - If true, returns image bytes. Otherwise, returns the content metadata.
	//  in: query
	//  type: boolean
	//  required: false
	IncludeImage bool `query:"image" json:"image"`
}

func (h HTTP) view(c echo.Context) error {
	user := c.Get("current_user").(models.User)

	r := new(viewContentReq)
	r.ContentID = c.Param("id")
	if err := c.Bind(r); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	content, imgBytes, err := h.svc.Get(c, user.ID.Hex(), r.ContentID, r.ProjectID, r.DatasetID, r.IncludeImage)
	if err != nil {
		return err
	}

	if r.IncludeImage {
		buf := bytes.NewBuffer(imgBytes)

		c.Response().Header().Set("Content-Type", content.ContentType)
		c.Response().Header().Set("Content-Length", strconv.Itoa(len(buf.Bytes())))

		io.Copy(c.Response().Writer, buf)

		return c.NoContent(200)
	}

	return c.JSON(http.StatusOK, content)
}

// Content model sample response
// swagger:response sampleContentResp
type sampleContentResp struct {
	// in:body
	Body struct {
		Content []models.Content `json:"content"`
	}
}

// Content sample request
// swagger:parameters sampleContentReq
type sampleContentReq struct {
	//	ID of project
	//  in: query
	//  type: string
	//  required: true
	ProjectID string `query:"project_id" json:"project_id" validate:"required"`
	//	Number of results to return
	//  in: query
	//  schema:
	//    type: integer
	//    minimum: 1
	//    maximum: 1000
	//  required: true
	Count int `query:"count" json:"count" validate:"required,min=1,max=1000"`
	//	Filter annotated content (boolean)
	//  in: query
	//  type: boolean
	//  required: false
	FilterAnnotated string `query:"filter_annotated" json:"filter_annotated" validate:"omitempty,boolean"`
}

func (h HTTP) sample(c echo.Context) error {
	r := new(sampleContentReq)

	if err := c.Bind(r); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)
	filterAnnotated, _ := strconv.ParseBool(r.FilterAnnotated) // intentionally ignored

	content, err := h.svc.Sample(c, user.ID.Hex(), r.ProjectID, r.Count, filterAnnotated)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	resp := sampleContentResp{struct {
		Content []models.Content "json:\"content\""
	}{content}}

	return c.JSON(http.StatusOK, resp.Body)
}

// Delete content request
// swagger:parameters deleteContentReq
type DeleteContentReq struct {
	// in: body
	Body struct {
		//	Project ID
		ProjectID string `json:"project_id" validate:"required"`
		//	Content IDs
		ContentIDs []string `json:"content_ids" validate:"required"`
	}
}

func (h HTTP) delete(c echo.Context) error {
	user := c.Get("current_user").(models.User)

	req := new(DeleteContentReq).Body
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	if err := h.svc.Delete(c, user.ID.Hex(), req.ProjectID, req.ContentIDs); err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.NoContent(http.StatusOK)
}

// Content query request
// swagger:parameters queryContentReq
//
//lint:ignore U1000 ignore, used for swagger spec
type queryContentReq struct {
	// in:body
	Body struct {
		models.QueryReq `json:"query"`
	}
}

// Content query response
// swagger:response queryContentResp
type queryContentResp struct {
	// in: body
	Body struct {
		Content []models.Content `json:"content"`
		Page    int              `json:"page"`
		Count   int64            `json:"count"`
	}
}

func (h HTTP) query(c echo.Context) error {
	var req models.QueryReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)

	content, count, err := h.svc.Query(c, user.ID.Hex(), req.Transform())
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	resp := queryContentResp{struct {
		Content []models.Content "json:\"content\""
		Page    int              "json:\"page\""
		Count   int64            "json:\"count\""
	}{content, req.Page, count}}

	return c.JSON(http.StatusOK, resp.Body)
}
