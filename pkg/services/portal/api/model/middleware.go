/*
 * File: middleware.go
 * Project: model
 * File Created: Monday, 2nd August 2021 2:00:36 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package model

import (
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func Noop(ctx echo.Context) (err error) {
	ctx.String(
		http.StatusNotImplemented,
		"No op handler should never be reached!",
	)

	return err
}

func ACAOHeaderOverwriteMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		ctx.Response().Before(func() {
			setResponseACAOHeaderFromRequest(*ctx.Request(), *ctx.Response())
		})
		return next(ctx)
	}
}

func setResponseACAOHeaderFromRequest(req http.Request, resp echo.Response) {
	resp.Header().Set(echo.HeaderAccessControlAllowOrigin, req.Header.Get(echo.HeaderOrigin))
	resp.Header().Set(echo.HeaderAccessControlAllowHeaders, "*")
}

func singleTargetBalancer(url *url.URL) middleware.ProxyBalancer {
	targetURL := []*middleware.ProxyTarget{
		{
			URL: url,
		},
	}
	return middleware.NewRoundRobinBalancer(targetURL)
}
