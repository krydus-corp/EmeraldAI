/*
 * File: swagger.go
 * Project: swagger
 * File Created: Wednesday, 21st September 2022 1:52:13 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package swaggerui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:generate go run generate.go

//go:embed embed
var swagfs embed.FS

// Handler returns a handler that will serve a self-hosted Swagger UI with your spec embedded
func Handler() http.Handler {
	// render the index template with the proper spec name inserted
	http.FileServer(http.FS(swagfs))
	static, err := fs.Sub(swagfs, "embed")
	if err != nil {
		panic(err)
	}

	return http.FileServer(http.FS(static))
}
