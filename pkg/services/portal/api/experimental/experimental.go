// Package user contains user application services
package experimental

import (
	"github.com/labstack/echo/v4"
)

const (
	ExperimentalAPIVersion = "0.1.0"
)

// Experimental returns the experimental API of users
func (e Experimental) Version(c echo.Context) string {
	return ExperimentalAPIVersion
}
