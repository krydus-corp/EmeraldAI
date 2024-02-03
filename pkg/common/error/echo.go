/*
 * File: echo.go
 * Project: error
 * File Created: Sunday, 15th May 2022 1:20:27 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package error

import "github.com/labstack/echo/v4"

func EchoErr(e error, code int) *echo.HTTPError {
	err, ok := e.(*echo.HTTPError)
	if !ok {
		return echo.NewHTTPError(code, e.Error())
	}
	return err
}
