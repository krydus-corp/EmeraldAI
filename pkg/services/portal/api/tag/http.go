package tag

import (
	"fmt"
	"net/http"

	errs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/error"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"

	"github.com/labstack/echo/v4"
)

// HTTP represents user http service
type HTTP struct {
	svc Service
}

func NewHTTP(svc Service, r *echo.Group) {
	h := HTTP{svc}
	ur := r.Group("/tags")

	// swagger:operation POST /v1/tags tags createTagReq
	// ---
	// summary: Create a new tag.
	// description: Creates a new tag for the current user.
	// security:
	// - Bearer: []
	// consumes:
	//  - application/json
	// produces:
	//  - application/json
	// responses:
	//   "200":
	//     "schema":
	//      "$ref": "#/definitions/Tag"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.POST("", h.create)

	// swagger:operation GET /v1/tags tags listTagsReq
	// ---
	// summary: Returns list of tags.
	// description: Returns list of tags for the current user and specified project.
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
	//     "$ref": "#/responses/listTagsResp"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("", h.list)

	// swagger:operation GET /v1/tags/{Id} tags getTagReq
	// ---
	// summary: Returns a single tag.
	// description: Returns a single tag by its ID.
	// security:
	// - Bearer: []
	// produces:
	//  - application/json
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of tag
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "schema":
	//      "$ref": "#/definitions/Tag"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("/:id", h.view)

	// swagger:operation PATCH /v1/tags/{Id} tags updateTagReq
	// ---
	// summary: Updates tag information.
	// description: Updates tag information -> name, property.
	// security:
	// - Bearer: []
	// consumes:
	//  - application/json
	// produces:
	//  - application/json
	// responses:
	//   "200":
	//     "schema":
	//      "$ref": "#/definitions/Tag"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.PATCH("/:id", h.update)

	// swagger:operation POST /v1/tags/query tags queryTagReq
	// ---
	// summary: Query tags.
	// description: |
	//   Query tags based on specified filter(s).
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
	//  "$ref": "#/definitions/queryTagReq"
	// responses:
	//   "200":
	//     "$ref": "#/responses/queryTagResp"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.POST("/query", h.query)

	// swagger:operation GET /v1/tags/properties tags getTagPropertiesReq
	// ---
	// summary: Returns all properties for tags.
	// description: Returns all properties for tags associated with a project & dataset.
	// security:
	// - Bearer: []
	// produces:
	//  - application/json
	// schema:
	//  "$ref": "#/definitions/getTagPropertiesReq"
	// responses:
	//   "200":
	//     "$ref": "#/responses/tagPropertyResp"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("/properties", h.properties)

	// swagger:operation DELETE /v1/tags/{Id} tags deleteTagReq
	// ---
	// summary: Deletes a tag.
	// description: Deletes a tag with requested ID.
	// security:
	// - Bearer: []
	// schema:
	//  "$ref": "#/definitions/deleteTagReq"
	// responses:
	//   "200":
	//     "$ref": "#/responses/ok"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.DELETE("/:id", h.delete)
}

// Tag create request
// swagger:parameters createTagReq
type createTagReq struct {
	// in: body
	Body struct {
		// Tag name
		TagName string `json:"name" validate:"required"`
		// Tag Properties
		Properties []string `json:"properties,omitempty" validate:"omitempty"`
		// Project ID
		ProjectID string `json:"project_id" validate:"required"`
		// Dataset ID
		DatasetID string `json:"dataset_id" validate:"required"`
	}
}

func (h HTTP) create(c echo.Context) error {
	r := new(createTagReq).Body
	if err := c.Bind(&r); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)
	tagModel := models.NewTag(user.ID.Hex(), r.ProjectID, r.DatasetID, r.TagName, r.Properties)

	tag, err := h.svc.Create(c, tagModel)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, tag)
}

// Tag list response
// swagger:response listTagsResp
type listTagsResp struct {
	// in:body
	Body struct {
		Tags  []models.Tag `json:"tags"`
		Page  int          `json:"page"`
		Count int64        `json:"count"`
	}
}

type listTagsReq struct {
	models.PaginationReq
	DatasetID string `json:"dataset_id" query:"dataset_id" validate:"required"`
}

func (h HTTP) list(c echo.Context) error {
	var req listTagsReq

	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)

	result, count, err := h.svc.List(c, user.ID.Hex(), req.DatasetID, req.Transform())
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	resp := listTagsResp{struct {
		Tags  []models.Tag "json:\"tags\""
		Page  int          "json:\"page\""
		Count int64        "json:\"count\""
	}{result, req.Page, count}}

	return c.JSON(http.StatusOK, resp.Body)
}

func (h HTTP) view(c echo.Context) error {
	user := c.Get("current_user").(models.User)
	tagid := c.Param("id")

	tag, err := h.svc.View(c, user.ID.Hex(), tagid)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, tag)
}

// Tag update request
// swagger:parameters updateTagReq
type updateTagReq struct {
	// ID of tag
	// in: path
	// type: string
	// required: true
	Id string `param:"Id" validate:"required"`
	// in: body
	Body struct {
		Name     string   `json:"name,omitempty" validate:"omitempty"`
		Property []string `json:"property,omitempty" validate:"omitempty,unique"`
	}
}

func (h HTTP) update(c echo.Context) error {
	user := c.Get("current_user").(models.User)
	tagid := c.Param("id")

	req := new(updateTagReq).Body
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	tag, err := h.svc.Update(c, Update{
		UserID:   user.ID.Hex(),
		TagID:    tagid,
		Name:     req.Name,
		Property: req.Property,
	})

	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, tag)
}

// Tag delete request
// swagger:parameters deleteTagReq
type deleteTagReq struct {
	// ID of tag
	// in: path
	// required: true
	Id string `param:"id" validate:"required"`
}

func (h HTTP) delete(c echo.Context) error {
	user := c.Get("current_user").(models.User)

	var req deleteTagReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	if err := h.svc.Delete(c, user.ID.Hex(), req.Id); err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": fmt.Sprintf("Tag %s deleted", req.Id)})
}

// Tag query request
// swagger:parameters queryTagReq
//
//lint:ignore U1000 ignore, used for swagger spec
type queryTagReq struct {
	// in:body
	Body struct {
		models.QueryReq `json:"query"`
	}
}

// Tag query response
// swagger:response queryTagResp
type queryTagResp struct {
	// in: body
	Body struct {
		Tags  []models.Tag `json:"tags"`
		Page  int          `json:"page"`
		Count int64        `json:"count"`
	}
}

func (h HTTP) query(c echo.Context) error {
	var req models.QueryReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)

	tags, count, err := h.svc.Query(c, user.ID.Hex(), req.Transform())
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	resp := queryTagResp{struct {
		Tags  []models.Tag "json:\"tags\""
		Page  int          "json:\"page\""
		Count int64        "json:\"count\""
	}{tags, req.Page, count}}

	return c.JSON(http.StatusOK, resp.Body)
}

// Tag property response
// swagger:response tagPropertyResp
type tagPropertyResp struct {
	// in: body
	Body struct {
		Properties []string `json:"properties"`
	}
}

// Tag properties request
// swagger:parameters getTagPropertiesReq
type getTagPropertiesReq struct {
	// ID of dataset
	// in: query
	DatasetID string `json:"dataset_id" query:"dataset_id" validate:"required"`
}

func (h HTTP) properties(c echo.Context) error {
	var req getTagPropertiesReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)

	properties, err := h.svc.Properties(c, user.ID.Hex(), req.DatasetID)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	resp := tagPropertyResp{struct {
		Properties []string "json:\"properties\""
	}{properties}}

	return c.JSON(http.StatusOK, resp.Body)
}
