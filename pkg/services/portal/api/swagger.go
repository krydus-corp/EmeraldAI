// Package classification EmeraldAI
//
// Emerald AI Portal Documentation.
//
//	    Schemes: https
//	    BasePath: /
//	    Version: 1.0.0
//
//	    Consumes:
//	    - application/json
//		   - application/txt
//		   - application/multipartform
//
//	    Produces:
//	    - application/json
//		   - application/txt
//
//	    SecurityDefinitions:
//	        Bearer:
//	            type: apiKey
//	            name: Authorization
//	            in: header
//
// swagger:meta
package api

// success response
// swagger:response ok
// description: Success response
//
//lint:ignore U1000 ignore, used for swagger spec
type SwaggOKResp struct {
	// in:body
	Body struct {
		Message string `json:"message"`
	}
}

// error response
// swagger:response err
// description: Error response
// copies echo.HTTPError
//
//lint:ignore U1000 ignore, used for swagger spec
type SwaggErrResp struct {
	// in:body
	Body struct {
		Message  string      `json:"message"`
		Code     int         `json:"code"`
		Internal interface{} `json:"internal"`
	}
}
