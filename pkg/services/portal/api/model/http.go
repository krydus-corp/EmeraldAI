package model

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	errs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/error"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
)

const (
	WEBSOCKET_NORMAL_CLOSURE = 1000
)

// HTTP represents model http service
type HTTP struct {
	svc Service
}

func NewHTTP(svc Service, r *echo.Group) {
	h := HTTP{svc}
	ur := r.Group("/models")

	// swagger:operation POST /v1/models models createModelReq
	// ---
	// summary: Creates new model.
	// description: Creates new model.
	// security:
	// - Bearer: []
	// consumes:
	//  - application/json
	// produces:
	//  - application/json
	// responses:
	//   "200":
	//     "schema":
	//      "$ref": "#/definitions/Model"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.POST("", h.create)

	// swagger:operation GET /v1/models models listModelReq
	// ---
	// summary: Returns list of models for the current user.
	// description: Returns list of projects for the current user.
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
	//     "$ref": "#/responses/listModelResp"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("", h.list)

	// swagger:operation GET /v1/models/{Id} models getModelReq
	// ---
	// summary: Gets a model.
	// description: |
	//   Gets a single model by it's ID.
	//   If the optional websocket parameter is specified, a websocket connection will be attempted.
	// security:
	// - Bearer: []
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of model
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
	//      "$ref": "#/definitions/Model"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("/:id", h.view)

	// swagger:operation PATCH /v1/models/{Id} models updateModelReq
	// ---
	// summary: Update a model.
	// description: |
	//   Updates a model.
	//   =============================
	//   **Key**: `name`
	//   **Description**: Model name.
	//   **Value**:  Any string.
	//
	//   **Key**: `parameters.resource.max_runtime_seconds`
	//   **Description**: Specifies a limit to how long a model hyperparameter training job can run.
	//   **Value**:  Any integer in the range [0, 86400]
	//
	//   **Key**: `parameters.resource.maximum_retry_attempts`
	//   **Description**: The number of times to retry the job if it fails due to an internal server error.
	//   **Value**: Any integer in the range [0, 10].
	//
	//   **Key**: `parameters.resource.max_number_training_jobs`
	//   **Description**: Max number of train jobs.
	//   **Value**: Any integer in the range [0, 100].
	//
	//   **Key**:`parameters.resource.instance_type`
	//   **Description**: AWS instance type to train on.
	//   *Value*: Any of the following:
	//       "ml.m4.xlarge"
	//       "ml.m4.2xlarge"
	//       "ml.m4.4xlarge"
	//       "ml.m4.10xlarge"
	//       "ml.m4.16xlarge"
	//       "ml.g4dn.xlarge"
	//       "ml.g4dn.2xlarge"
	//       "ml.g4dn.4xlarge"
	//       "ml.g4dn.8xlarge"
	//       "ml.g4dn.12xlarge"
	//       "ml.g4dn.16xlarge"
	//       "ml.m5.large"
	//       "ml.m5.xlarge"
	//       "ml.m5.2xlarge"
	//       "ml.m5.4xlarge"
	//       "ml.m5.12xlarge"
	//       "ml.m5.24xlarge"
	//       "ml.c4.xlarge"
	//       "ml.c4.2xlarge"
	//       "ml.c4.4xlarge"
	//       "ml.c4.8xlarge"
	//       "ml.p2.xlarge"
	//       "ml.p2.8xlarge"
	//       "ml.p2.16xlarge"
	//       "ml.p3.2xlarge"
	//       "ml.p3.8xlarge"
	//       "ml.p3.16xlarge"
	//       "ml.p3dn.24xlarge"
	//       "ml.p4d.24xlarge"
	//       "ml.c5.xlarge"
	//       "ml.c5.2xlarge"
	//       "ml.c5.4xlarge"
	//       "ml.c5.9xlarge"
	//       "ml.c5.18xlarge"
	//       "ml.c5n.xlarge"
	//       "ml.c5n.2xlarge"
	//       "ml.c5n.4xlarge"
	//       "ml.c5n.9xlarge"
	//       "ml.c5n.18xlarge"
	//       "ml.g5.xlarge"
	//       "ml.g5.2xlarge"
	//       "ml.g5.4xlarge"
	//       "ml.g5.8xlarge"
	//       "ml.g5.16xlarge"
	//       "ml.g5.12xlarge"
	//       "ml.g5.24xlarge"
	//       "ml.g5.48xlarge"
	//
	//   **Key**: `hyperparameters`
	//   **Description**: Hyperperameters is a mapping of tunable training parameter to it's value.</br>The acceptable parameters are based on model architecture.
	//   **Value**: See Description of the options below:
	//
	//   <h2>Object Detection Parameters</h2>
	//   **Key**: `hyperparameters.epochs`
	//   **Description**: The number of training epochs.
	//   **Value**:  Any integer in the range [0, 1000].
	//
	//   **Key**: `hyperparameters.lr_scheduler_step`
	//   **Description**: The epochs at which to reduce the learning rate.</br>The learning rate is reduced by lr_scheduler_factor at epochs listed in a comma-delimited string: "epoch1, epoch2, ...".
	//   **Value**: Any string. For example, if the value is set to "10, 20" and the lr_scheduler_factor is set to 1/2, then the learning rate is halved after 10th epoch and then halved again after 20th epoch.
	//
	//   **Key**: `hyperparameters.lr_scheduler_factor`
	//   **Description**: The ratio to reduce learning rate.</br>Used in conjunction with the lr_scheduler_step parameter defined as lr_new = lr_old * lr_scheduler_factor.
	//   **Value**: Any float in the range (0:1)
	//
	//   **Key**: `hyperparameters.overlap_threshold`
	//   **Description**: The evaluation overlap threshold.
	//   **Value**: Any float in the range (0:1)
	//
	//   **Key**: `hyperparameters.nms_threshold`
	//   **Description**: he non-maximum suppression threshold.
	//   **Value**: Any float in the range (0:1]
	//
	//   **Key**: `hyperparameters.learning_rate`
	//   **Description**: The initial learning rate.
	//   **Value**: Any float in the range (0:1]
	//
	//   **Key**: `hyperparameters.weight_decay`
	//   **Description**: The weight decay coefficient for sgd and rmsprop. Ignored for other optimizers.
	//
	//   <h2>Classification Parameters</h2>
	//   **Key**: `hyperparameters.num_layers`
	//   **Description**: Number of layers for the network.For data with large image size (for example, 224x224 - like ImageNet), we suggest selecting the number of layers from the set [18, 34, 50, 101, 152, 200].</br>For data with small image size (for example, 28x28 - like CIFAR), we suggest selecting the number of layers from the set [20, 32, 44, 56, 110].
	//   **Value**: Any positive integer in [18, 34, 50, 101, 152, 200] or [20, 32, 44, 56, 110]
	//
	//   **Key**: `hyperparameters.epochs`
	//   **Description**: The number of training epochs.
	//   **Value**:  Any integer in the range [0, 1000].
	//
	//   **Key**: `hyperparameters.lr_scheduler_step`
	//   **Description**: The epochs at which to reduce the learning rate.</br>The learning rate is reduced by lr_scheduler_factor at epochs listed in a comma-delimited string: "epoch1, epoch2, ...".
	//   **Value**: Any string. For example, if the value is set to "10, 20" and the lr_scheduler_factor is set to 1/2, then the learning rate is halved after 10th epoch and then halved again after 20th epoch.
	//
	//   **Key**: `hyperparameters.lr_scheduler_factor`
	//   **Description**: The ratio to reduce learning rate.</br>Used in conjunction with the lr_scheduler_step parameter defined as lr_new = lr_old * lr_scheduler_factor.
	//   **Value**: Any float in the range (0:1)
	//
	// security:
	//   - Bearer: []
	// schema:
	//   "$ref": "#/definitions/updateModelReq"
	// responses:
	//   "200":
	//     "schema":
	//      "$ref": "#/definitions/Model"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.PATCH("/:id", h.update)

	// swagger:operation POST /v1/models/{Id} models deleteModelReq
	// ---
	// summary: Deletes a model
	// description: Deletes a single model by its ID and all associated information -> jobs and exports.
	// security:
	// - Bearer: []
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of model
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

	// swagger:operation POST /v1/models/{Id}/train models trainModelReq
	// ---
	// summary: Trains a model.
	// description: |
	//   Trains a single model by its associated id. This is an asynchronous request and will return immediately if the request is well formed.
	//   Status of the train job can be viewed by looking up the id of the passed in model.
	// security:
	// - Bearer: []
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of model
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "schema":
	//      "$ref": "#/definitions/Model"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.POST("/:id/train", h.train)

	// swagger:operation POST /v1/models/{Id}/deploy models deployModelReq
	// ---
	// summary: Deploys a model.
	// description: |
	//   Deploys a single model by its associated id. This is an asynchronous request and will return immediately if the request is well formed.
	//   Status of the deployment job can be viewed by looking up the id of the passed in model.
	// security:
	// - Bearer: []
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of model
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "schema":
	//      "$ref": "#/definitions/Model"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.POST("/:id/deploy", h.deployment)

	// swagger:operation DELETE /v1/models/{Id}/deploy models deployDeleteModelReq
	// ---
	// summary: Deletes a model deployment.
	// description: |
	//   Deleted a model deployment by its associated model id. This is an asynchronous request and will return immediately if the request is well formed.
	//   Status of the deployment can be viewed by looking up the id of the passed in model.
	// security:
	// - Bearer: []
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of model
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "schema":
	//      "$ref": "#/definitions/Model"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.DELETE("/:id/deploy", h.deleteDeployment)

	// swagger:operation POST /v1/models/{Id}/inference/realtime models realtimeModelReq
	// ---
	// summary: Synchronous inference for select number of content.
	// description: |
	//   Run inference synchronously for select number of content.
	//   Content can be specified by id usign the `content_id` query parameter, image(s) provided as a form-file,
	//   or single image send via an octet-stream in the request body. The correct `content-type`` header must be
	//   set for multipart/form-data or application/octet-stream mime types.
	// security:
	// - Bearer: []
	// consumes:
	//  - multipart/form-data
	//  - application/octet-stream
	// produces:
	//  - application/json
	// parameters:
	// - name: Id
	//   in: path
	//   description: ID of model
	//   type: string
	//   required: true
	// - name: files
	//   in: formData
	//   description: Image(s) to predict
	//   type: file
	//   required: false
	// - name: threshold
	//   in: query
	//   description: Confidence threshold to limit return prediction.
	//   type: number
	//   format: float
	//   minimum: 0.0
	//   maximum: 1.0
	//   default: 0.85
	// - name: heatmap
	//   in: query
	//   description: Include heatmap in the response.
	//   type: boolean
	//   default: false
	//   required: false
	// responses:
	//   "200":
	//      "$ref": "#/responses/ok"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.POST("/:id/inference/realtime",
		Noop,
		ACAOHeaderOverwriteMiddleware,
		middleware.ProxyWithConfig(middleware.ProxyConfig{
			Balancer:  singleTargetBalancer(&url.URL{Scheme: "https", Host: "model:8081"}),
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, // Internal service uses a self signed cert
		}),
	)

	// swagger:operation POST /v1/models/{Id}/inference/batch models batchModelReq
	// ---
	// summary: Run inference on entire project using the selected model.
	// description: Run inference on entire project using the model associated with a project and user. This is an asynchronous request and will return immediately if the request is well formed.
	// security:
	// - Bearer: []
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of model
	//   type: string
	//   required: true
	// - name: thumbnail_size
	//   in: query
	//   description: thumbnail size for predictions. oneof 100, 200, 640.
	//   type: integer
	//   required: false
	// responses:
	//   "200":
	//     "$ref": "#/responses/ok"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.POST("/:id/inference/batch", h.createBatch)

	// swagger:operation POST /v1/models/query models queryModelReq
	// ---
	// summary: Query models
	// description: |
	//   Query models based on specified filter(s).
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
	// schema:
	//  "$ref": "#/definitions/queryModelReq"
	// responses:
	//   "200":
	//     "$ref": "#/responses/queryModelResp"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.POST("/query", h.query)

	// swagger:operation POST /v1/models/{Id} models deleteModelReq
	// ---
	// summary: Deletes a model
	// description: Deletes a single model by its ID and all associated information -> jobs and exports.
	// security:
	// - Bearer: []
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of model
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
}

// Model create request
// swagger:parameters createModelReq
type createModelReq struct {
	// in: body
	Body struct {
		// Name of project
		Name string `json:"name" validate:"required"`
		// Project ID
		ProjectID string `json:"project_id" validate:"required"`
		// Preprocessors
		Preprocessors models.Preprocessors `json:"preprocessors,omitempty"`
		// Augmentations
		Augmentations models.Augmentations `json:"augmentations,omitempty"`
	}
}

func (h HTTP) create(c echo.Context) error {
	req := new(createModelReq).Body
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)
	retModel, err := h.svc.Create(c, user.ID.Hex(), req.ProjectID, req.Name, req.Preprocessors, req.Augmentations)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, retModel)
}

func (h HTTP) view(c echo.Context) error {
	user := c.Get("current_user").(models.User)
	modelid := c.Param("id")
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
			// Get training status
			model, err := h.svc.View(c, user.ID.Hex(), modelid)
			if err != nil {
				log.Infof("error getting training status; err=%s", err.Error())
				if err := ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(WEBSOCKET_NORMAL_CLOSURE, err.Error())); err != nil {
					log.Errorf("failed sending websocket close message; err=%s", err.Error())
				}
				return nil
			}

			// Write
			if err := ws.WriteJSON(model); err != nil && err != io.EOF {
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

	model, err := h.svc.View(c, user.ID.Hex(), modelid)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, model)
}

type listModelReq struct {
	models.PaginationReq
	ProjectID string `json:"project_id" query:"project_id" validate:"required"`
}

// Model list response
// swagger:response listModelResp
type listModelResp struct {
	// in:body
	Body struct {
		Models []models.Model `json:"models"`
		Page   int            `json:"page"`
		Count  int64          `json:"count"`
	}
}

func (h HTTP) list(c echo.Context) error {
	var req listModelReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)

	result, count, err := h.svc.List(c, user.ID.Hex(), req.ProjectID, req.Transform())
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	resp := listModelResp{struct {
		Models []models.Model "json:\"models\""
		Page   int            "json:\"page\""
		Count  int64          "json:\"count\""
	}{result, req.Page, count}}

	return c.JSON(http.StatusOK, resp.Body)
}

// Model query request
// swagger:parameters queryModelReq
//
//lint:ignore U1000 ignore, used for swagger spec
type queryModelReq struct {
	// in:body
	Body struct {
		models.QueryReq `json:"query"`
		Page            int `json:"page"`
	}
}

// Model query response
// swagger:response queryModelResp
type queryModelResp struct {
	// in: body
	Body struct {
		Models []models.Model `json:"models"`
		Page   int            `json:"page"`
		Count  int64          `json:"count"`
	}
}

func (h HTTP) query(c echo.Context) error {
	var req models.QueryReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)

	modelResp, count, err := h.svc.Query(c, user.ID.Hex(), req.Transform())
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	resp := queryModelResp{struct {
		Models []models.Model "json:\"models\""
		Page   int            "json:\"page\""
		Count  int64          "json:\"count\""
	}{modelResp, req.Page, count}}

	return c.JSON(http.StatusOK, resp.Body)
}

// Model update request
// swagger:parameters updateModelReq
type updateModelReq struct {
	// in:body
	Body struct {
		// Name of project
		Name string `json:"name,omitempty" validate:"omitempty"`
		// Preprocessors
		Preprocessors models.Preprocessors `json:"preprocessors,omitempty"`
		// Augmentations
		Augmentations models.Augmentations `json:"augmentations,omitempty"`
		// Train Parameters
		Parameters models.TrainParameters `json:"parameters,omitempty"`
	}

	// ID of model
	// in: path
	// type: string
	// required: true
	Id string `param:"Id" validate:"required"`
}

func (h HTTP) update(c echo.Context) error {
	modelId := c.Param("id")
	req := new(updateModelReq).Body
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)
	model, err := h.svc.Update(c, Update{
		ModelID:       modelId,
		UserID:        user.ID.Hex(),
		Name:          req.Name,
		Parameters:    req.Parameters,
		Preprocessing: req.Preprocessors,
		Augmentation:  req.Augmentations,
	})
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, model)
}

func (h HTTP) delete(c echo.Context) error {
	user := c.Get("current_user").(models.User)
	modelid := c.Param("id")

	if err := h.svc.Delete(c, user.ID.Hex(), modelid); err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": fmt.Sprintf("Model %s deleted", modelid)})
}

func (h HTTP) train(c echo.Context) error {
	// Required params
	userid := c.Request().Header.Get("userid")
	if userid == "" {
		return c.JSON(401, echo.ErrUnauthorized)
	}
	modelid := c.Param("id")
	if modelid == "" {
		return c.JSON(400, echo.NewHTTPError(400, "model `id` required"))
	}

	model, err := h.svc.Train(c, userid, modelid)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, model)
}

func (h HTTP) deployment(c echo.Context) error {
	// Required params
	userid := c.Request().Header.Get("userid")
	if userid == "" {
		return c.JSON(401, echo.ErrUnauthorized)
	}
	modelid := c.Param("id")
	if modelid == "" {
		return c.JSON(400, echo.NewHTTPError(400, "model `id` required"))
	}

	err := h.svc.Deploy(c, userid, modelid)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Deploying"})
}

func (h HTTP) deleteDeployment(c echo.Context) error {
	// Required params
	userid := c.Request().Header.Get("userid")
	if userid == "" {
		return c.JSON(401, echo.ErrUnauthorized)
	}
	modelid := c.Param("id")
	if modelid == "" {
		return c.JSON(400, echo.NewHTTPError(400, "model `id` required"))
	}

	err := h.svc.DeleteDeployment(c, userid, modelid)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Deleting"})
}

// Model create request
// swagger:parameters createModelReq
type createBatchReq struct {
	// in: body
	Body struct {
		// Thumbnail Width and Height
		ThumbnailSize int `json:"thumbnail_size,omitempty" validate:"oneof=0 100 200 640"`
	}
}

func (h HTTP) createBatch(c echo.Context) error {
	// Required params
	userid := c.Request().Header.Get("userid")
	if userid == "" {
		return c.JSON(401, echo.ErrUnauthorized)
	}
	modelid := c.Param("id")
	if modelid == "" {
		return c.JSON(400, echo.NewHTTPError(400, "model `id` required"))
	}

	req := new(createBatchReq).Body
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	err := h.svc.CreateBatch(c, userid, modelid, req.ThumbnailSize)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Creating"})
}
