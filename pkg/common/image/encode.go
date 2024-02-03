// Package imager provides image processing utilities
/*
 * File: encode.go
 * Project: image
 * File Created: Saturday, 4th April 2020 9:46:49 pm
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
	"image/jpeg"
	"image/png"
	"net/http"
	"path"
	"strings"
)

const (
	// ContentTypeJPEG is the 'image/jpeg' MIME type
	ContentTypeJPEG = "image/jpeg"
	// ContentTypePNG is the 'image/png' MIME type
	ContentTypePNG = "image/png"
)

// checkSupportedContentType is a helper function for checking supported content types for conversion
func checkSupportedContentType(contentType string) bool {
	switch contentType {
	case ContentTypeJPEG, ContentTypePNG:
		return true
	}
	return false
}

// ConvertImgType converts an image from one MIME type to another.
// A valid MIME type must be passed in for the image to be converted.
// Reference https://tools.ietf.org/html/rfc2045 and https://tools.ietf.org/html/rfc2046 for valid MIME type.
//
// Currently, this function only supports the following MIME type: image/jpeg, image/png
func ConvertImgType(imgBytes []byte, dstMimeType string) ([]byte, error) {
	srcMimeType := http.DetectContentType(imgBytes)
	if srcMimeType == dstMimeType {
		return imgBytes, nil
	}

	var err error
	var encodedImg *bytes.Buffer

	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, err
	}

	switch dstMimeType {
	case ContentTypeJPEG:
		encodedImg, err = encodeImageToJPEG(img)
	case ContentTypePNG:
		encodedImg, err = encodeImageToPNG(img)
	default:
		return nil, fmt.Errorf("unsupported destination MIME type '%s'", dstMimeType)
	}

	if err != nil {
		return nil, err
	}

	return encodedImg.Bytes(), nil
}

// encodeImageToJPEG is a helper function for encoding an image to JPEG format
func encodeImageToJPEG(img image.Image) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, img, nil)

	return buf, err
}

// encodeImageToPNG is a helper function for encoding an image to PNG format
func encodeImageToPNG(img image.Image) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	err := png.Encode(buf, img)

	return buf, err
}

// updatePathExtension is a helper function for updating the extension of a filepath
// based on a given content type.
func updatePathExtension(filepath, contentType string) string {
	return strings.TrimSuffix(filepath, path.Ext(filepath)) + "." + strings.Split(contentType, "/")[1]
}
