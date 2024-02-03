/*
 * File: resize_test.go
 * Project: image
 * File Created: Tuesday, 14th April 2020 7:55:47 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package image

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResize(t *testing.T) {
	const (
		OriginalWidth  = 400
		OriginalHeight = 200
	)

	jpegImg, _ := newImage("image/jpeg", OriginalWidth, OriginalHeight)

	tests := map[string]struct {
		img        []byte
		convHeight int
		convWidth  int
		err        bool
	}{
		"JPEG Resize": {jpegImg, 100, 100, false},
	}

	for name, test := range tests {
		t.Logf("Running test %s", name)

		newImg, err := ResizeImage(test.img, test.convWidth, test.convHeight)
		if err != nil {
			t.Errorf("Unexpected error resizing image; error=%s", err.Error())
			continue
		}

		stats, err := GetStats(newImg)
		if err != nil {
			t.Errorf("Unexpected error getting image stats; error=%s", err.Error())
			continue
		}

		t.Logf("Resized image from (%d, %d) -> (%d, %d)", OriginalHeight, OriginalWidth, test.convHeight, test.convWidth)
		assert.Equal(t, stats.Height, test.convHeight)
		assert.Equal(t, stats.Width, test.convWidth)
	}
}
