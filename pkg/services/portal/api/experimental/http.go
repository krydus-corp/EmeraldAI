/*
 * File: http.go
 * Project: experimental
 * File Created: Friday, 9th June 2023 12:12:54 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package experimental

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// HTTP represents the experimental service
type HTTP struct {
	svc Service
}

// NewHTTP creates new user http service
func NewHTTP(svc Service, r *echo.Group) {
	h := HTTP{svc}
	ur := r.Group("/experimental")

	// swagger:operation GET /v1/experimental experimental getExperimentalVersionReq
	// ---
	// summary: Experimental API version.
	// description: |
	//   Experimental API version.
	// security:
	// - Bearer: []
	// consumes:
	//  - application/json
	// produces:
	//  - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/ok"
	//   "400":
	//     "$ref": "#/responses/err"
	//   "401":
	//     "$ref": "#/responses/err"
	//   "500":
	//     "$ref": "#/responses/err"
	ur.GET("", h.version)
}

// version is a method for getting the experimental API version.
func (h HTTP) version(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"version": h.svc.Version(c)})
}
