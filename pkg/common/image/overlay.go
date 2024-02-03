/*
 * File: overlay.go
 * Project: image
 * File Created: Sunday, 16th April 2023 6:12:29 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package image

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

func AddOverlay(i image.Image, overlay image.Image, transparency float64) image.Image {
	mask := image.NewUniform(color.Alpha{uint8(math.Floor(255 * transparency))})

	canvas := image.NewRGBA(overlay.Bounds())
	draw.Draw(canvas, canvas.Bounds(), overlay, image.Point{0, 0}, draw.Src)

	draw.DrawMask(canvas, canvas.Bounds(), i, image.Point{0, 0}, mask, image.Point{0, 0}, draw.Over)

	return canvas
}
