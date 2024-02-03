/*
 * File: http.go
 * Project: export
 * File Created: Tuesday, 14th September 2021 7:12:40 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package export

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	errs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/error"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
)

const (
	WEBSOCKET_NORMAL_CLOSURE = 1000
)

// HTTP represents export http service
type HTTP struct {
	svc Service
}

func NewHTTP(svc Service, r *echo.Group) {
	h := HTTP{svc}
	ur := r.Group("/exports")

	// swagger:operation POST /v1/exports exports createExportReq
	// ---
	// summary: Creates an export.
	// description: Creates an export for the given user and project.
	// security:
	// - Bearer: []
	// consumes:
	//  - application/json
	// produces:
	//  - application/json
	// schema:
	//  "$ref": "#/definitions/createExportReq"
	// responses:
	//   "200":
	//     "schema":
	//      "$ref": "#/definitions/Export"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.POST("", h.create)

	// swagger:operation GET /v1/exports exports listExportsReq
	// ---
	// summary: List exports.
	// description: List exports associated with user and project.
	// security:
	// - Bearer: []
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
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
	//     "$ref": "#/responses/listExportsResp"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("", h.listExports)

	// swagger:operation POST /v1/exports/query exports queryExportReq
	// ---
	// summary: Query exports.
	// description: |
	//   Query exports based on specified filter(s).
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
	//  "$ref": "#/definitions/queryExportReq"
	// responses:
	//   "200":
	//     "$ref": "#/responses/queryExportResp"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.POST("/query", h.query)

	// swagger:operation GET /v1/exports/{Id} exports viewExportReq
	// ---
	// summary: Get export.
	// description: |
	//   Get export by it's ID.
	//   If the optional websocket parameter is specified, a websocket connection will be attempted.
	// security:
	// - Bearer: []
	// produces:
	//  - application/json
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of export
	//   type: string
	//   required: true
	// - name: websocket
	//   in: query
	//   description: Establish a websocket connection
	//   type: boolean
	//   required: false
	// responses:
	//   "200":
	//     "schema":
	//      "$ref": "#/definitions/Export"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("/:id", h.viewExport)

	// swagger:operation DELETE /v1/exports/{Id} exports deleteExportReq
	// ---
	// summary: Deletes an export.
	// description: Deletes an export with requested ID.
	// security:
	// - Bearer: []
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of export
	//   type: string
	//   required: true
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

	// swagger:operation GET /v1/exports/{Id}/metadata exports getExportMetadataReq
	// ---
	// summary: Retrieves export metadata.
	// description: Retrieves the metadata from an export with requested ID.
	// security:
	// - Bearer: []
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of export
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ok"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("/:id/metadata", h.metadata)

	// swagger:operation GET /v1/exports/{Id}/content exports getExportContentReq
	// ---
	// summary: Retrieves export content.
	// description: Retrieves the content presigned URL links for an export with requested ID.
	// security:
	// - Bearer: []
	// schema:
	//  "$ref": "#/definitions/getExportContentReq"
	// responses:
	//   "200":
	//     "$ref": "#/responses/ok"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("/:id/content", h.content)
}

// Export create request
// swagger:parameters createExportReq
type createExportReq struct {
	// Export name
	// in: query
	// required: true
	Name string `json:"name" validate:"required"`
	// Type of export
	// in: query
	// required: true
	ExportType string `json:"export_type" validate:"required,oneof=project dataset model"`
	// Entity ID e.g. project, dataset, or model
	// in: query
	// required: true
	ID string `json:"id"`
}

func (h HTTP) create(c echo.Context) error {
	req := new(createExportReq)
	if err := c.Bind(req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)

	export, err := h.svc.Create(c, user.ID.Hex(), *req)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, export)
}

func (h HTTP) viewExport(c echo.Context) error {
	user := c.Get("current_user").(models.User)
	exportid := c.Param("id")
	wsString := c.QueryParam("websocket")

	ws, _ := strconv.ParseBool(wsString)

	// If establishing a websocket connection,
	if ws {
		var upgrader = websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}
		ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			log.Infof("error upgrading connection; err=%s", err.Error())
			if err := ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(WEBSOCKET_NORMAL_CLOSURE, err.Error())); err != nil {
				log.Errorf("failed sending websocket close message; err=%s", err.Error())
			}
			return nil
		}
		defer ws.Close()

		for {
			export, err := h.svc.View(c, user.ID.Hex(), exportid)
			if err != nil {
				log.Infof("error getting export; err=%s", err.Error())
				if err := ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(WEBSOCKET_NORMAL_CLOSURE, err.Error())); err != nil {
					log.Errorf("failed sending websocket close message; err=%s", err.Error())
				}
				return nil
			}
			if err := ws.WriteJSON(export); err != nil && err != io.EOF {
				if !errors.Is(err, syscall.EPIPE) {
					log.Infof("error writing response; err=%s", err.Error())
					if err := ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(WEBSOCKET_NORMAL_CLOSURE, err.Error())); err != nil {
						log.Errorf("failed sending websocket close message; err=%s", err.Error())
					}
				}
				return nil
			}

			time.Sleep(1 * time.Second)
		}
	}

	export, err := h.svc.View(c, user.ID.Hex(), exportid)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, export)
}

type listExportsReq struct {
	models.PaginationReq
}

// Export list response
// swagger:response listExportsResp
type listResponse struct {
	// in:body
	Body struct {
		Exports []models.Export `json:"exports"`
		Page    int             `json:"page"`
		Count   int64           `json:"count"`
	}
}

func (h HTTP) listExports(c echo.Context) error {
	var req listExportsReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)
	result, count, err := h.svc.List(c, user.ID.Hex(), req.Transform())
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	resp := listResponse{struct {
		Exports []models.Export "json:\"exports\""
		Page    int             "json:\"page\""
		Count   int64           "json:\"count\""
	}{result, req.Page, count}}

	return c.JSON(http.StatusOK, resp.Body)
}

// Export query request
// swagger:parameters queryExportReq
//
//lint:ignore U1000 ignore, used for swagger spec
type queryExportReq struct {
	// in:body
	Body struct {
		models.QueryReq `json:"query"`
	}
}

// Export query response
// swagger:response queryExportResp
type queryExportResp struct {
	// in: body
	Body struct {
		Exports []models.Export `json:"exports"`
		Page    int             `json:"page"`
		Count   int64           `json:"count"`
	}
}

func (h HTTP) query(c echo.Context) error {
	var req models.QueryReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)

	exports, count, err := h.svc.Query(c, user.ID.Hex(), req.Transform())
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	resp := queryExportResp{struct {
		Exports []models.Export "json:\"exports\""
		Page    int             "json:\"page\""
		Count   int64           "json:\"count\""
	}{exports, req.Page, count}}

	return c.JSON(http.StatusOK, resp.Body)
}

func (h HTTP) delete(c echo.Context) error {
	user := c.Get("current_user").(models.User)
	id := c.Param("id")

	if err := h.svc.Delete(c, user.ID.Hex(), id); err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": fmt.Sprintf("Export %s deleted", id)})
}

func (h HTTP) metadata(c echo.Context) error {
	user := c.Get("current_user").(models.User)
	id := c.Param("id")

	metadataJsonBytes, err := h.svc.GetMetadata(c, user.ID.Hex(), id)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	buf := bytes.NewBuffer(metadataJsonBytes)

	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().Header().Set("Content-Length", strconv.Itoa(len(buf.Bytes())))
	c.Response().Header().Set("Content-Disposition", "attachment; filename=metadata.json")

	io.Copy(c.Response().Writer, buf)

	return c.NoContent(http.StatusTemporaryRedirect)
}

// Export content request
// swagger:parameters getExportContentReq
type getExportContentReq struct {
	// ID of export
	// in: path
	// type: string
	// required: true
	Id string `param:"Id" validate:"required"`
	// Content URI
	//
	// in: query
	// type: string
	// required: true
	URI string `json:"uri" query:"uri" validate:"required"`
	// Automatically begin download
	//
	// in: query
	// type: boolean
	// required: false
	Download string `json:"download" query:"download"`
	// Expire time in minutes. Defaults to 5min.
	//
	// in: query
	// type: integer
	// min: 1
	// max: 1440
	// required: false
	Expire int `json:"expire" query:"expire" validate:"number,min=1,max=1440"`
}

func (h HTTP) content(c echo.Context) error {
	var req getExportContentReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)
	id := c.Param("id")
	download, _ := strconv.ParseBool(req.Download)
	if req.Expire == 0 {
		req.Expire = 5
	}

	url, err := h.svc.GetContent(c, user.ID.Hex(), id, req.URI, req.Expire)
	if err != nil {
		return err
	}

	if !download {
		return c.JSON(http.StatusOK, map[string]string{"URL": url})
	}

	c.Response().Header().Set("Content-Type", "application/zip")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", path.Base(url)))
	c.Response().Header().Set("Access-Control-Allow-Origin", (c.Request()).Header.Get("Origin"))
	c.Response().Header().Set("Access-Control-Allow-Credentials", "true")
	c.Response().Header().Set("Vary", "Origin")

	http.Redirect(c.Response().Writer, c.Request(), url, http.StatusTemporaryRedirect)

	return c.NoContent(http.StatusTemporaryRedirect)
}
