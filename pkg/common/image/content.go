/*
 * File: content.go
 * Project: image
 * File Created: Thursday, 7th October 2021 6:14:29 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package image

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
)

// newImage is a function for providing a test image
func newImage(mimeType string, width, height int) ([]byte, error) {
	// Create an 100 x 50 image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Draw a red dot at (2, 3)
	img.Set(2, 3, color.RGBA{255, 0, 0, 255})

	var b = new(bytes.Buffer)
	var err error

	switch mimeType {
	case "image/jpeg":
		err = jpeg.Encode(b, img, nil)
	case "image/png":
		err = png.Encode(b, img)
	default:
		return nil, fmt.Errorf("unsupported MIME type '%s'", mimeType)
	}

	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
