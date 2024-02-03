/*
 * File: http.go
 * Project: api
 * File Created: Tuesday, 13th July 2021 2:10:57 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package api

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	common "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	errs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/error"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	server "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/server"
	sage "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/sage"
)

// HTTP represents user http service
type HTTP struct {
	svc Service
}

// NewHTTP creates new user http service
func NewHTTP(svc Service, r *echo.Group) {
	h := HTTP{svc}
	ur := r.Group("/models")

	ur.POST("/:id/inference/realtime", h.realtimeInference)
}

func (h HTTP) realtimeInference(c echo.Context) error {

	type multipartFiles struct {
		Files []*multipart.FileHeader `form:"files"`
	}

	// Required params
	userid := c.Request().Header.Get("userid")
	if userid == "" {
		return c.JSON(401, echo.ErrUnauthorized)
	}
	modelid := c.Param("id")
	if modelid == "" {
		return c.JSON(400, echo.NewHTTPError(400, "model `id` required"))
	}
	// Optional params
	threshold, err := strconv.ParseFloat(c.QueryParam("threshold"), 64)
	if err != nil {
		threshold = sage.Const_DefaultRealtimePredictionConfidenceThreshold
	}
	heatmap, err := strconv.ParseBool(c.QueryParam("heatmap"))
	if err != nil {
		heatmap = false
	}

	realtimeInferenceReq := realtimeInferenceReq{
		userID:              userid,
		modelID:             modelid,
		confidenceThreshold: threshold,
		files:               map[string][]byte{},
		octetStream:         []byte{},
		heatmap:             heatmap,
	}

	// Content is in the body
	if c.Request().ContentLength > 0 {
		switch contentType := c.Request().Header.Get("content-type"); {
		// Checking for octet stream
		case contentType == "application/octet-stream":
			defer c.Request().Body.Close()
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, c.Request().Body); err != nil {
				return c.JSON(500, echo.NewHTTPError(500, "unable to read octet-stream"))
			}

			realtimeInferenceReq.octetStream = buf.Bytes()

			results, err := h.svc.RealtimeInference(c, realtimeInferenceReq)
			if err != nil {
				err := errs.EchoErr(err, 500)
				return c.JSON(err.Code, err)
			}
			return c.JSONPretty(http.StatusOK, results, " ")
		// Checking for form files
		case strings.HasPrefix(contentType, "multipart/form-data"):
			c.Echo().Binder = server.NewBindFile(c.Echo().Binder)

			var req multipartFiles
			if err := c.Bind(&req); err != nil {
				return c.JSON(500, echo.NewHTTPError(500, "error parsing form data"))
			}

			for _, f := range req.Files {
				body, err := common.ReadFile(f)
				if err != nil {
					log.Infof("unable to parse multipart.File; err=%s", err.Error())
					continue
				}
				if body != nil {
					realtimeInferenceReq.files[f.Filename] = body
				}
			}

			results, err := h.svc.RealtimeInference(c, realtimeInferenceReq)
			if err != nil {
				err := errs.EchoErr(err, 500)
				return c.JSON(err.Code, err)
			}
			return c.JSONPretty(http.StatusOK, results, " ")
		default:
			return c.JSON(400, echo.NewHTTPError(400, "unexpected 'Content-Type' header with non-empty request body"))
		}
	}

	return c.JSON(400, echo.NewHTTPError(400, "'Content-Length' header is 0; 'Content-Type' must be either multipart/form-data or application/octet-stream"))
}
