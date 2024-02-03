package dataset

import (
	"net/http"

	errs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/error"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/labstack/echo/v4"
)

// HTTP represents user http service
type HTTP struct {
	svc Service
}

func NewHTTP(svc Service, r *echo.Group) {
	h := HTTP{svc}
	ur := r.Group("/datasets")

	// swagger:operation GET /v1/datasets/{Id} datasets getDatasetReq
	// ---
	// summary: Returns a single dataset.
	// description: Returns a single dataset by ID.
	// security:
	// - Bearer: []
	// produces:
	//  - application/json
	// parameters:
	// - name: Id
	//   in: path
	//   description: id of dataset
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/datasetResp"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("/:id", h.view)

	// swagger:operation PATCH /v1/datasets/{Id} datasets updateDatasetReq
	// ---
	// summary: Updates a dataset.
	// description: Updates a dataset, including the following fields - split.
	// security:
	// - Bearer: []
	// consumes:
	//  - application/json
	// produces:
	//  - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/datasetResp"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.PATCH("/:id", h.update)
}

// swagger:response datasetResp
// forces generation of swagger model
//
//lint:ignore U1000 ignore, used for swagger spec
type datasetResp struct {
	// in: body
	Body struct {
		Dataset models.Dataset `json:"dataset"`
	}
}

func (h HTTP) view(c echo.Context) error {
	user := c.Get("current_user").(models.User)
	datasetid := c.Param("id")

	dataset, err := h.svc.View(c, user.ID.Hex(), datasetid)
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	resp := datasetResp{struct {
		Dataset models.Dataset "json:\"dataset\""
	}{
		*dataset,
	}}

	return c.JSON(http.StatusOK, resp.Body)
}

// Dataset update request
// swagger:parameters updateDatasetReq
type updateDatasetReq struct {
	// ID of dataset
	// in: path
	// type: string
	// required: true
	Id string `param:"Id" validate:"required"`
	// in: body
	Body struct {
		Split struct {
			Train      float64 `json:"train" validate:"number,min=0,max=1"`
			Validation float64 `json:"validation" validate:"number,min=0,max=1"`
			Test       float64 `json:"test" validate:"number,min=0,max=1"`
		} `json:"split,omitempty"`
	}
}

func (h HTTP) update(c echo.Context) error {
	idStr := c.Param("id")
	user := c.Get("current_user").(models.User)

	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	req := new(updateDatasetReq).Body
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.NewHTTPError(400, err.Error()))
	}

	dataset, err := h.svc.Update(c, models.Dataset{
		ID:     id,
		UserID: user.ID.Hex(),
		Split: struct {
			Train      float64 "json:\"train\" bson:\"train\""
			Validation float64 "json:\"validation\" bson:\"validation\""
			Test       float64 "json:\"test\" bson:\"test\""
		}(req.Split),
	})
	if err != nil {
		err := errs.EchoErr(err, 500)
		return c.JSON(err.Code, err)
	}

	resp := datasetResp{struct {
		Dataset models.Dataset "json:\"dataset\""
	}{
		*dataset,
	}}

	return c.JSON(http.StatusOK, resp.Body)
}
