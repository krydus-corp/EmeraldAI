/*
 * File: encode_test.go
 * Project: image
 * File Created: Sunday, 5th April 2020 3:36:47 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package image

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncode(t *testing.T) {
	const (
		Width  = 400
		Height = 200
	)

	jpegImg, _ := newImage("image/jpeg", Width, Height)
	pngImg, _ := newImage("image/png", Width, Height)

	tests := map[string]struct {
		img        []byte
		conversion string
		err        bool
	}{
		"PNG to JPEG":         {pngImg, "image/jpeg", false},
		"JPEG to PNG":         {jpegImg, "image/png", false},
		"JPEG to Unsupported": {jpegImg, "image/svg", true},
	}

	for name, test := range tests {
		t.Logf("Running test %s", name)

		imgResult, err := ConvertImgType(test.img, test.conversion)
		if err != nil {
			// Not expecting an error on this test
			if !test.err {
				t.Errorf("failed converting image to MIME type '%s'; %s", test.conversion, err.Error())
			}
			continue
		}

		mimeType := http.DetectContentType(imgResult)

		assert.Equal(t, test.conversion, mimeType)
	}
}
