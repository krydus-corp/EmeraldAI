package prediction

import (
	"io"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	errs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/error"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
)

// HTTP represents prediction http service
type HTTP struct {
	svc Service
}

// NewHTTP creates new prediction http service
func NewHTTP(svc Service, r *echo.Group) {
	h := HTTP{svc}
	ur := r.Group("/predictions")

	// swagger:operation POST /v1/predictions/query predictions queryPredictionsReq
	// ---
	// summary: Query predictions.
	// description: |
	//   Query predictions based on specified filter(s).
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
	//  "$ref": "#/definitions/queryPredictionsReq"
	// responses:
	//   "200":
	//     "$ref": "#/responses/queryPredictionsResp"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.POST("/query", h.query)

	// swagger:operation GET /v1/predictions/statistics predictions predictionStatisticsReq
	// ---
	// summary: Get prediction statistics.
	// description: Get prediction total and class counts for a given model.
	// security:
	// - Bearer: []
	// produces:
	//  - application/json
	// parameters:
	// - name: model_id
	//   in: query
	//   type: string
	//   required: true
	// - name: uncertainty_threshold
	//   in: query
	//   type: integer
	//   required: false
	// - name: tag_id
	//   in: query
	//   type: array
	//   items:
	//     type: string
	//   required: false
	// responses:
	//   "200":
	//     "$ref": "#/responses/predictionsStatisticsResp"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("/statistics", h.statistics)

	// swagger:operation GET /v1/predictions/sample predictions predictionsReq
	// ---
	// summary: Get sample of predictions in model predictions.
	// description: |
	//   Get sample of predictions in model predictions.
	//   Optionally, predictions can be filtered by tagid and a confidence threshold.
	// security:
	// - Bearer: []
	// produces:
	//  - application/json
	// parameters:
	// - name: model_id
	//   in: query
	//   type: string
	//   required: true
	// - name: uncertainty_threshold
	//   in: query
	//   type: integer
	//   required: true
	// - name: tag_id
	//   in: query
	//   type: array
	//   items:
	//     type: string
	//   required: false
	// - name: sample_count
	//   in: query
	//   type: integer
	//   required: false
	// - name: thumbnail_size
	//   in: query
	//   type: integer
	//   required: false
	// responses:
	//   "200":
	//     "$ref": "#/responses/samplePredictionsResp"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("/sample", h.predictions)

	// swagger:operation GET /v1/predictions/{Id}/heatmap predictions heatmapReq
	// ---
	// summary: Get heatmap for a prediction.
	// description: |
	//   Get heatmap for a prediction.
	// security:
	// - Bearer: []
	// produces:
	//  - application/json
	// parameters:
	// - name: Id
	//   in: path
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ok"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("/:id/heatmap", h.heatmap)
}

// Predictions query request
// swagger:parameters queryPredictionsReq
//
//lint:ignore U1000 ignore, used for swagger spec
type queryPredictionsReq struct {
	// in:body
	Body struct {
		models.QueryReq `json:"query"`
		Page            int `json:"page"`
	}
}

// Predictions query response
// swagger:response queryPredictionsResp
type queryPredictionsResp struct {
	// in: body
	Body struct {
		Predictions []models.Prediction `json:"predictions"`
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

	predictions, count, err := h.svc.Query(c, user.ID.Hex(), req.Transform())
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	resp := queryPredictionsResp{struct {
		Predictions []models.Prediction "json:\"predictions\""
		Page        int                 "json:\"page\""
		Count       int64               "json:\"count\""
	}{predictions, req.Page, count}}

	return c.JSON(http.StatusOK, resp.Body)
}

type predictionCountStatisticsReq struct {
	ModelID              string   `json:"model_id" query:"model_id" validate:"required"`
	UncertaintyThreshold float64  `json:"uncertainty_threshold" query:"uncertainty_threshold" validate:"required,number,min=0,max=1"`
	TagID                []string `json:"tag_id" query:"tag_id"`
}

// Prediction stats response
// swagger:response predictionsStatisticsResp
type predictionsStatisticsResp struct {
	// in: body
	Body struct {
		Statistics
	}
}

func (h HTTP) statistics(c echo.Context) error {
	var req predictionCountStatisticsReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)

	stats, err := h.svc.Statistics(c, user.ID.Hex(), req.ModelID, req.UncertaintyThreshold, req.TagID...)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	return c.JSON(http.StatusOK, predictionsStatisticsResp{struct{ Statistics }{*stats}})
}

type predictionsReq struct {
	ModelID              string   `json:"model_id" query:"model_id" validate:"required"`
	UncertaintyThreshold float64  `json:"uncertainty_threshold" query:"uncertainty_threshold" validate:"required,number,min=0,max=1"`
	TagID                []string `json:"tag_id" query:"tag_id"`
	SampleCount          int      `json:"sample_count" query:"sample_count" validate:"number,min=1,max=100"`
	ThumbnailSize        int      `json:"thumbnail_size" query:"thumbnail_size" validate:"omitempty,number,oneof=100 200 640"`
}

// Prediction sample response
// swagger:response samplePredictionsResp
type samplePredictionsResp struct {
	// in: body
	Body struct {
		Predictions []models.Prediction `json:"predictions"`
	}
}

func (h HTTP) predictions(c echo.Context) error {
	var req predictionsReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)

	predictions, err := h.svc.Predictions(c, user.ID.Hex(), req.ModelID, req.UncertaintyThreshold, req.SampleCount, req.ThumbnailSize, req.TagID...)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	resp := samplePredictionsResp{struct {
		Predictions []models.Prediction "json:\"predictions\""
	}{predictions}}

	return c.JSON(http.StatusOK, resp.Body)
}

type heatmapReq struct {
	PredictionID string `param:"id" validate:"required"`
}

func (h HTTP) heatmap(c echo.Context) error {
	var req heatmapReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	user := c.Get("current_user").(models.User)

	heatmap, err := h.svc.Heatmap(c, user.ID.Hex(), req.PredictionID)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	c.Response().Header().Set("Content-Type", "image/jpeg")
	c.Response().Header().Set("Content-Length", strconv.Itoa(len(heatmap.Bytes())))

	io.Copy(c.Response().Writer, heatmap)

	return c.NoContent(200)
}
