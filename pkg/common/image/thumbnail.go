/*
 * File: thumbnail.go
 * Project: image
 * File Created: Sunday, 9th May 2021 12:22:42 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"net/http"

	"github.com/disintegration/imaging"
)

const (
	BoundingBoxWidth                = 5 // pixel width of bounding boxes.
	BoundingBoxDefaultThumbnailSize = 100
)

type BoundingBox struct {
	Xmin      int
	Xmax      int
	Ymin      int
	Ymax      int
	ClassName string
}

// Thumbnail is a function that scales the image up or down using the specified resample filter,
// crops it to the specified width and hight and returns a base64 encoded string of the transformed image.
func Thumbnail(imgBytes []byte, width, height int) (string, int, error) {
	// Decode image
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return "", 0, err
	}

	// Resize the image
	dstImg := imaging.Thumbnail(img, width, height, imaging.Lanczos)

	// Encode back to original format
	var encodedImg *bytes.Buffer

	contentType := http.DetectContentType(imgBytes)
	switch contentType {
	case ContentTypeJPEG:
		encodedImg, err = encodeImageToJPEG(dstImg)
	case ContentTypePNG:
		encodedImg, err = encodeImageToPNG(dstImg)
	default:
		return "", 0, fmt.Errorf("unsupported MIME type '%s'", contentType)
	}

	if err != nil {
		return "", 0, err
	}

	sEnc := base64.StdEncoding.EncodeToString(encodedImg.Bytes())

	return sEnc, len([]byte(sEnc)), nil
}

func ThumbnailBoundingBox(imgBytes []byte, width, height int, boundingBoxes []BoundingBox) (string, int, error) {
	orig, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return "", 0, err
	}
	// convert as usable image
	b := orig.Bounds()
	img := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(img, img.Bounds(), orig, b.Min, draw.Src)

	myRectangles := make(map[string][]image.Rectangle)
	for _, box := range boundingBoxes {
		myRectangle := image.Rect(
			box.Xmin,
			box.Ymin,
			box.Xmax,
			box.Ymax,
		)
		myRectangles[box.ClassName] = append(myRectangles[box.ClassName], myRectangle)
	}
	boundingBoxImage := addRectanglesToFace(img, myRectangles)

	// Resize the image
	dstImg := imaging.Thumbnail(boundingBoxImage, width, height, imaging.Lanczos)

	// Encode back to original format
	var encodedImg *bytes.Buffer

	contentType := http.DetectContentType(imgBytes)
	switch contentType {
	case ContentTypeJPEG:
		encodedImg, err = encodeImageToJPEG(dstImg)
	case ContentTypePNG:
		encodedImg, err = encodeImageToPNG(dstImg)
	default:
		return "", 0, fmt.Errorf("unsupported MIME type '%s'", contentType)
	}

	if err != nil {
		return "", 0, err
	}

	sEnc := base64.StdEncoding.EncodeToString(encodedImg.Bytes())

	return sEnc, len([]byte(sEnc)), nil
}

func drawRectangle(img draw.Image, color color.Color, x1, y1, x2, y2, width int) {
	for i := x1; i < x2; i++ {
		for j := width / 2; j >= 0; j-- {
			img.Set(i, y1, color)
			img.Set(i, y1+j, color)
			img.Set(i, y1-j, color)

			img.Set(i, y2, color)
			img.Set(i, y2+j, color)
			img.Set(i, y2-j, color)
		}
	}

	for i := y1; i <= y2; i++ {
		for j := width / 2; j >= 0; j-- {
			img.Set(x1, i, color)
			img.Set(x1+j, i, color)
			img.Set(x1-j, i, color)
			img.Set(x2, i, color)
			img.Set(x2+j, i, color)
			img.Set(x2-j, i, color)
		}
	}
}

func addRectanglesToFace(img draw.Image, rectangles map[string][]image.Rectangle) draw.Image {
	// TODO add option to pass in desired colors
	colorOptions := []color.RGBA{
		{0x00, 0xFF, 0x00, 0xFF}, // light green
		{245, 86, 39, 0xFF},      // orange
		{39, 245, 245, 0xFF},     // teal
		{245, 245, 39, 0xFF},     // yellow
		{245, 39, 86, 0xFF},      // red

	}

	i := 0
	for _, rectangles := range rectangles {
		if i > len(colorOptions)-1 {
			i = 0
		}
		for _, rectangle := range rectangles {
			min := rectangle.Min
			max := rectangle.Max
			drawRectangle(img, colorOptions[i], min.X, min.Y, max.X, max.Y, BoundingBoxWidth)
		}
		i += 1
	}

	return img
}
