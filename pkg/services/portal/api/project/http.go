package project

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"

	errs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/error"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/api/project/upload"
)

const (
	UPLOAD_TIMEOUT_SECONDS = 60 * 60 * 5 // 5 hours
	UPLOAD_RATE_LIMIT      = 100
)

// HTTP represents user http service
type HTTP struct {
	svc Service
}

func NewHTTP(svc Service, r *echo.Group) {
	h := HTTP{svc}
	ur := r.Group("/projects")

	// swagger:operation POST /v1/projects projects createProjectReq
	// ---
	// summary: Creates a new project.
	// description: Creates a new project for the specified user.
	// security:
	// - Bearer: []
	// consumes:
	//  - application/json
	// produces:
	//  - application/json
	// responses:
	//   "200":
	//     "schema":
	//      "$ref": "#/definitions/Project"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "409":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.POST("", h.create)

	// swagger:operation GET /v1/projects projects listProjectsReq
	// ---
	// summary: Returns list of projects.
	// description: Returns list of projects for the current user.
	// security:
	// - Bearer: []
	// consumes:
	// - application/json
	// produces:
	//  - application/json
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
	//     "$ref": "#/responses/listProjectsResp"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("", h.list)

	// swagger:operation GET /v1/projects/{Id} projects getProjectReq
	// ---
	// summary: Returns a single project.
	// description: Returns a single project by its ID.
	// security:
	// - Bearer: []
	// produces:
	//  - application/json
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of project
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "schema":
	//      "$ref": "#/definitions/Project"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("/:id", h.view)

	// swagger:operation POST /v1/projects/{Id}/profile projects getProjectProfileReq
	// ---
	// summary: Updates a project profile picture.
	// description: |
	//   Updates a project profile picture with requested ID.
	//   Uploaded file should be an image in JPEG or PNG format.
	//   The optional `random` parameter can be passed in to associate with a random image in the project.
	// security:
	// - Bearer: []
	// consumes:
	//  - multipart/form-data
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of project
	//   type: string
	//   required: true
	// - name: file
	//   in: formData
	//   description: Profile picture to upload.
	//   type: file
	//   required: false
	// - name: random
	//   in: query
	//   description: Associate with random image in project (optional)
	//   type: boolean
	//   required: false
	//   default: false
	// responses:
	//   "200":
	//     "$ref": "#/responses/ok"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.POST("/:id/profile", h.profile)

	// swagger:operation POST /v1/projects/{Id}/upload projects uploadProjectReq
	// ---
	// summary: Upload content to a project.
	// description: |
	//   Uploads content for the given user and project.
	//   Uploaded files must be `multipart/form-data` data.
	//   File(s) can be specified with key as `files`.
	//   This endpoint processes files in realtime. Once the request returns, all files are processed.
	//
	//   Example response:
	//   ```json
	//   {
	//       "labels_file": "labels.json",
	//        "errors": [
	//        {
	//            "Error": "UploadErrInvalidFormat",
	//            "InternalError": "",
	//            "Filename": "train.manifest",
	//            "Message": "The provided file format is not allowed. Please upload a JPEG or PNG image"
	//        }
	//        ],
	//        "total_bytes": 3591,
	//        "total_files": 1,
	//        "total_files_failed": 1,
	//        "total_files_succeeded": 0,
	//        "total_files_duplicate": 0
	//   }
	//   ```
	// security:
	// - Bearer: []
	// consumes:
	//  - application/json
	// produces:
	//  - application/json
	//  - multipart/form-data
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of project
	//   type: string
	//   required: true
	// - name: labels_file
	//   in: query
	//   type: string
	//   required: false
	//   description: JSON file containing labels for the associated upload.
	// - name: files
	//   in: formData
	//   description: File(s) with data for the project
	//   type: file
	//   required: false
	// responses:
	//   "200":
	//     "$ref": "#/responses/uploadProjectResp"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.POST("/:id/upload", h.upload, middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Timeout Exceeded",
		Timeout:      time.Second * UPLOAD_TIMEOUT_SECONDS,
	}), middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(UPLOAD_RATE_LIMIT)))
	// swagger:operation PATCH /v1/projects/{Id} projects updateProjectReq
	// ---
	// summary: Updates a project.
	// description: Updates a project name, description, annotation_type, and license.
	// security:
	// - Bearer: []
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     "schema":
	//      "$ref": "#/definitions/Project"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.PATCH("/:id", h.update)

	// swagger:operation DELETE /v1/projects/{Id} projects deleteProjectReq
	// ---
	// summary: Deletes a project.
	// description: Deletes a single project by its ID and all associated information -> tags, content, models, annotations, exports.
	// security:
	// - Bearer: []
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of project
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ok"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.DELETE("/:id", h.delete)

	// swagger:operation POST /v1/projects/query projects queryProjectReq
	// ---
	// summary: Query projects.
	// description: |
	//   Query projects based on specified filter(s).
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
	// produces:
	//  - application/json
	// schema:
	//  "$ref": "#/definitions/queryProjectReq"
	// responses:
	//   "200":
	//     "$ref": "#/responses/queryProjectResp"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.POST("/query", h.query)
}

// Create project request
// swagger:parameters createProjectReq
type createProjectReq struct {
	// in: body
	Body struct {
		// Name of project
		ProjectName string `json:"name" query:"name" validate:"required"`
		// Description of project
		ProjectDescription string `json:"description" query:"description"`
		// Annotation type of project
		ProjectAnnotation string `json:"annotation_type" query:"annotation_type" validate:"required,oneof=classification bounding_box"`
		// License type of project
		ProjectLicense string `json:"license" query:"license"`
	}
}

func (h HTTP) create(c echo.Context) error {
	r := new(createProjectReq).Body
	if err := c.Bind(&r); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)

	projectModel, err := models.NewProject(user.ID.Hex(), r.ProjectName, r.ProjectDescription, r.ProjectLicense, r.ProjectAnnotation)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	datasetModel := models.NewDataset(user.ID.Hex(), projectModel.ID.Hex())
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	projectModel.DatasetID = datasetModel.ID.Hex()

	project, err := h.svc.Create(c, projectModel, datasetModel)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, project)
}

func (h HTTP) view(c echo.Context) error {
	projectid := c.Param("id")
	user := c.Get("current_user").(models.User)

	project, err := h.svc.View(c, user.ID.Hex(), projectid)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, project)
}

// Projects list response
// swagger:response listProjectsResp
type listProjectsResp struct {
	// in: body
	Body struct {
		Projects []models.Project `json:"projects"`
		Page     int              `json:"page"`
		Count    int64            `json:"count"`
	}
}

func (h HTTP) list(c echo.Context) error {
	var req models.PaginationReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)

	result, count, err := h.svc.List(c, user.ID.Hex(), req.Transform())
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	resp := listProjectsResp{struct {
		Projects []models.Project "json:\"projects\""
		Page     int              "json:\"page\""
		Count    int64            "json:\"count\""
	}{result, req.Page, count}}

	return c.JSON(http.StatusOK, resp.Body)
}

func (h HTTP) delete(c echo.Context) error {
	projectid := c.Param("id")
	user := c.Get("current_user").(models.User)

	// TODO: turn this into a status-able job that can be checked by the user
	err := h.svc.Delete(c, user.ID.Hex(), projectid)
	if err != nil {
		log.Errorf("error during delete project operation; err=%s", err.Error())
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": fmt.Sprintf("Project %s deleted", projectid)})
}

func (h HTTP) profile(c echo.Context) error {
	random := c.QueryParam("random")
	projectid := c.Param("id")
	user := c.Get("current_user").(models.User)

	randomBool, err := strconv.ParseBool(random)
	if err != nil {
		randomBool = false
	}

	var filePtr *multipart.File
	var filename = ""

	if !randomBool {
		maxSize := int64(1024000 * 1000 * .5) // allow only 500MB of file size for now

		err := c.Request().ParseMultipartForm(maxSize)
		if err != nil {
			return c.JSON(415, echo.NewHTTPError(415, err.Error()))
		}

		file, headers, err := c.Request().FormFile("file")
		if err != nil {
			return c.JSON(400, echo.NewHTTPError(400, err.Error()))
		}
		defer file.Close()

		filePtr = &file
		filename = headers.Filename
	}

	filename, err = h.svc.Profile(c, user.ID.Hex(), projectid, filename, filePtr)
	if err != nil {
		log.Errorf("could not upload file; err=s", err.Error())
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, filename)
}

// Project query request
// swagger:parameters queryProjectReq
//
//lint:ignore U1000 ignore, used for swagger spec
type queryProjectReq struct {
	// in: body
	Body struct {
		models.QueryReq `json:"query"`
	}
}

// Project query response
// swagger:response queryProjectResp
type queryProjectResp struct {
	// in: body
	Body struct {
		Projects []models.Project `json:"projects"`
		Page     int              `json:"page"`
		Count    int64            `json:"count"`
	}
}

func (h HTTP) query(c echo.Context) error {
	var req models.QueryReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)

	projects, count, err := h.svc.Query(c, user.ID.Hex(), req.Transform())
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	resp := queryProjectResp{struct {
		Projects []models.Project "json:\"projects\""
		Page     int              "json:\"page\""
		Count    int64            "json:\"count\""
	}{projects, req.Page, count}}

	return c.JSON(http.StatusOK, resp.Body)
}

// Update project request
// swagger:parameters updateProjectReq
type updateProjectReq struct {
	// ID of project
	// in: path
	// type: string
	// required: true
	Id string `param:"Id" validate:"required"`
	// in: body
	Body struct {
		// New project name
		ProjectName string `json:"name" query:"name"`
		// New project license
		ProjectLicense *string `json:"license" query:"license"`
		// New project description
		ProjectDescription *string `json:"description" query:"description"`
	}
}

func (h HTTP) update(c echo.Context) error {
	projectid := c.Param("id")
	user := c.Get("current_user").(models.User)

	req := new(updateProjectReq).Body
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	// Update project
	project, err := h.svc.Update(c, Update{
		ProjectID:   projectid,
		UserID:      user.ID.Hex(),
		Name:        req.ProjectName,
		Description: req.ProjectDescription,
		License:     req.ProjectLicense,
	})

	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, project)
}

// Upload response
// swagger:response uploadProjectResp
//
//lint:ignore U1000 ignore, used for swagger spec
type uploadProjectResp struct {
	// in:body
	Body struct {
		*upload.Report
	}
}

func (h HTTP) upload(c echo.Context) error {
	projectid := c.Param("id")
	user := c.Get("current_user").(models.User)
	labelsFile := c.QueryParam("labels_file")

	// Limit max request body size
	var maxAggregateRequestBody int64 = 1024 * 1024 * 1000 * 5 // 5GB
	c.Request().Body = http.MaxBytesReader(c.Response().Writer, c.Request().Body, maxAggregateRequestBody)
	if err := c.Request().ParseMultipartForm(maxAggregateRequestBody); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, errors.Wrapf(err, "The uploaded file(s) are too big. Please choose an file that's less than %dGB in size", maxAggregateRequestBody/1024*1024*1000)))
	}

	form, err := c.MultipartForm()
	if err != nil {
		return c.JSON(415, echo.NewHTTPError(415, err.Error()))
	}

	if _, ok := form.File["files"]; !ok {
		return c.JSON(400, echo.NewHTTPError(400, "form param 'files' required"))
	}

	req := CreateUploadReq{UserID: user.ID.Hex(), ProjectID: projectid, LabelsFile: labelsFile, Files: form.File["files"]}
	report, err := h.svc.Upload(c, req)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, *report)
}
