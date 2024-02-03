/*
 * File: heatmap.go
 * Project: scratch
 * File Created: Sunday, 16th April 2023 2:02:55 pm
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
	"sync"
)

// Classic is a color scheme that goes through a variety of colors.
var Classic []color.Color

func init() {
	Classic = []color.Color{
		color.RGBA{R: 0xff, G: 0xed, B: 0xed, A: 0xff},
		color.RGBA{R: 0xff, G: 0xe0, B: 0xe0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xd1, B: 0xd1, A: 0xff},
		color.RGBA{R: 0xff, G: 0xc1, B: 0xc1, A: 0xff},
		color.RGBA{R: 0xff, G: 0xb0, B: 0xb0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x9f, B: 0x9f, A: 0xff},
		color.RGBA{R: 0xff, G: 0x8e, B: 0x8e, A: 0xff},
		color.RGBA{R: 0xff, G: 0x7e, B: 0x7e, A: 0xff},
		color.RGBA{R: 0xff, G: 0x6e, B: 0x6e, A: 0xff},
		color.RGBA{R: 0xff, G: 0x5e, B: 0x5e, A: 0xff},
		color.RGBA{R: 0xff, G: 0x51, B: 0x51, A: 0xff},
		color.RGBA{R: 0xff, G: 0x43, B: 0x43, A: 0xff},
		color.RGBA{R: 0xff, G: 0x38, B: 0x38, A: 0xff},
		color.RGBA{R: 0xff, G: 0x2e, B: 0x2e, A: 0xff},
		color.RGBA{R: 0xff, G: 0x25, B: 0x25, A: 0xff},
		color.RGBA{R: 0xff, G: 0x1d, B: 0x1d, A: 0xff},
		color.RGBA{R: 0xff, G: 0x17, B: 0x17, A: 0xff},
		color.RGBA{R: 0xff, G: 0x12, B: 0x12, A: 0xff},
		color.RGBA{R: 0xff, G: 0xe, B: 0xe, A: 0xff},
		color.RGBA{R: 0xff, G: 0xb, B: 0xb, A: 0xff},
		color.RGBA{R: 0xff, G: 0x8, B: 0x8, A: 0xff},
		color.RGBA{R: 0xff, G: 0x6, B: 0x6, A: 0xff},
		color.RGBA{R: 0xff, G: 0x5, B: 0x5, A: 0xff},
		color.RGBA{R: 0xff, G: 0x3, B: 0x3, A: 0xff},
		color.RGBA{R: 0xff, G: 0x2, B: 0x2, A: 0xff},
		color.RGBA{R: 0xff, G: 0x2, B: 0x2, A: 0xff},
		color.RGBA{R: 0xff, G: 0x1, B: 0x1, A: 0xff},
		color.RGBA{R: 0xff, G: 0x1, B: 0x1, A: 0xff},
		color.RGBA{R: 0xff, G: 0x0, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x0, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x0, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x0, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x0, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x0, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x0, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x0, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x1, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x4, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x6, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xa, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xe, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x12, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x16, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x1a, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x1f, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x24, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x29, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x2d, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x33, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x39, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x3e, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x44, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x4a, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x51, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x56, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x5d, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x63, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x69, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x6f, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x76, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x7c, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x83, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x89, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x90, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x96, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0x9c, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xa3, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xa9, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xaf, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xb5, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xbb, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xc0, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xc6, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xcb, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xd0, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xd5, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xda, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xde, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xe3, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xe8, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xeb, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xee, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xf2, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xf5, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xf7, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xfa, B: 0x0, A: 0xff},
		color.RGBA{R: 0xff, G: 0xfb, B: 0x0, A: 0xff},
		color.RGBA{R: 0xfd, G: 0xfc, B: 0x0, A: 0xff},
		color.RGBA{R: 0xfa, G: 0xfc, B: 0x1, A: 0xff},
		color.RGBA{R: 0xf8, G: 0xfc, B: 0x2, A: 0xff},
		color.RGBA{R: 0xf4, G: 0xfc, B: 0x2, A: 0xff},
		color.RGBA{R: 0xf1, G: 0xfc, B: 0x3, A: 0xff},
		color.RGBA{R: 0xed, G: 0xfc, B: 0x3, A: 0xff},
		color.RGBA{R: 0xe9, G: 0xfc, B: 0x3, A: 0xff},
		color.RGBA{R: 0xe5, G: 0xfc, B: 0x4, A: 0xff},
		color.RGBA{R: 0xe1, G: 0xfc, B: 0x4, A: 0xff},
		color.RGBA{R: 0xdc, G: 0xfc, B: 0x5, A: 0xff},
		color.RGBA{R: 0xd8, G: 0xfc, B: 0x5, A: 0xff},
		color.RGBA{R: 0xd3, G: 0xfc, B: 0x6, A: 0xff},
		color.RGBA{R: 0xce, G: 0xfc, B: 0x7, A: 0xff},
		color.RGBA{R: 0xc9, G: 0xfc, B: 0x7, A: 0xff},
		color.RGBA{R: 0xc5, G: 0xfc, B: 0x8, A: 0xff},
		color.RGBA{R: 0xbf, G: 0xfb, B: 0x8, A: 0xff},
		color.RGBA{R: 0xb9, G: 0xf9, B: 0x9, A: 0xff},
		color.RGBA{R: 0xb4, G: 0xf7, B: 0x9, A: 0xff},
		color.RGBA{R: 0xae, G: 0xf6, B: 0xa, A: 0xff},
		color.RGBA{R: 0xa9, G: 0xf4, B: 0xb, A: 0xff},
		color.RGBA{R: 0xa4, G: 0xf2, B: 0xb, A: 0xff},
		color.RGBA{R: 0x9e, G: 0xf0, B: 0xc, A: 0xff},
		color.RGBA{R: 0x97, G: 0xee, B: 0xd, A: 0xff},
		color.RGBA{R: 0x92, G: 0xec, B: 0xe, A: 0xff},
		color.RGBA{R: 0x8c, G: 0xe9, B: 0xe, A: 0xff},
		color.RGBA{R: 0x86, G: 0xe7, B: 0xf, A: 0xff},
		color.RGBA{R: 0x80, G: 0xe4, B: 0x10, A: 0xff},
		color.RGBA{R: 0x7a, G: 0xe2, B: 0x11, A: 0xff},
		color.RGBA{R: 0x74, G: 0xdf, B: 0x12, A: 0xff},
		color.RGBA{R: 0x6e, G: 0xdd, B: 0x13, A: 0xff},
		color.RGBA{R: 0x69, G: 0xda, B: 0x14, A: 0xff},
		color.RGBA{R: 0x63, G: 0xd8, B: 0x15, A: 0xff},
		color.RGBA{R: 0x5d, G: 0xd6, B: 0x16, A: 0xff},
		color.RGBA{R: 0x58, G: 0xd3, B: 0x17, A: 0xff},
		color.RGBA{R: 0x52, G: 0xd1, B: 0x18, A: 0xff},
		color.RGBA{R: 0x4c, G: 0xcf, B: 0x19, A: 0xff},
		color.RGBA{R: 0x47, G: 0xcc, B: 0x1a, A: 0xff},
		color.RGBA{R: 0x42, G: 0xca, B: 0x1c, A: 0xff},
		color.RGBA{R: 0x3c, G: 0xc8, B: 0x1e, A: 0xff},
		color.RGBA{R: 0x37, G: 0xc6, B: 0x1f, A: 0xff},
		color.RGBA{R: 0x32, G: 0xc4, B: 0x21, A: 0xff},
		color.RGBA{R: 0x2d, G: 0xc2, B: 0x22, A: 0xff},
		color.RGBA{R: 0x28, G: 0xbf, B: 0x23, A: 0xff},
		color.RGBA{R: 0x24, G: 0xbe, B: 0x25, A: 0xff},
		color.RGBA{R: 0x1f, G: 0xbc, B: 0x27, A: 0xff},
		color.RGBA{R: 0x1b, G: 0xbb, B: 0x28, A: 0xff},
		color.RGBA{R: 0x17, G: 0xb9, B: 0x2b, A: 0xff},
		color.RGBA{R: 0x13, G: 0xb8, B: 0x2c, A: 0xff},
		color.RGBA{R: 0xf, G: 0xb7, B: 0x2e, A: 0xff},
		color.RGBA{R: 0xc, G: 0xb6, B: 0x30, A: 0xff},
		color.RGBA{R: 0x9, G: 0xb5, B: 0x33, A: 0xff},
		color.RGBA{R: 0x6, G: 0xb5, B: 0x35, A: 0xff},
		color.RGBA{R: 0x3, G: 0xb4, B: 0x37, A: 0xff},
		color.RGBA{R: 0x1, G: 0xb4, B: 0x39, A: 0xff},
		color.RGBA{R: 0x0, G: 0xb4, B: 0x3c, A: 0xff},
		color.RGBA{R: 0x0, G: 0xb4, B: 0x3e, A: 0xff},
		color.RGBA{R: 0x0, G: 0xb4, B: 0x41, A: 0xff},
		color.RGBA{R: 0x0, G: 0xb5, B: 0x44, A: 0xff},
		color.RGBA{R: 0x0, G: 0xb6, B: 0x46, A: 0xff},
		color.RGBA{R: 0x0, G: 0xb6, B: 0x4a, A: 0xff},
		color.RGBA{R: 0x0, G: 0xb7, B: 0x4d, A: 0xff},
		color.RGBA{R: 0x0, G: 0xb8, B: 0x50, A: 0xff},
		color.RGBA{R: 0x0, G: 0xb8, B: 0x54, A: 0xff},
		color.RGBA{R: 0x0, G: 0xba, B: 0x58, A: 0xff},
		color.RGBA{R: 0x0, G: 0xbb, B: 0x5c, A: 0xff},
		color.RGBA{R: 0x0, G: 0xbc, B: 0x5f, A: 0xff},
		color.RGBA{R: 0x0, G: 0xbe, B: 0x63, A: 0xff},
		color.RGBA{R: 0x0, G: 0xbf, B: 0x68, A: 0xff},
		color.RGBA{R: 0x0, G: 0xc1, B: 0x6c, A: 0xff},
		color.RGBA{R: 0x0, G: 0xc2, B: 0x70, A: 0xff},
		color.RGBA{R: 0x0, G: 0xc4, B: 0x74, A: 0xff},
		color.RGBA{R: 0x0, G: 0xc6, B: 0x78, A: 0xff},
		color.RGBA{R: 0x0, G: 0xc8, B: 0x7d, A: 0xff},
		color.RGBA{R: 0x0, G: 0xc9, B: 0x81, A: 0xff},
		color.RGBA{R: 0x0, G: 0xcb, B: 0x86, A: 0xff},
		color.RGBA{R: 0x0, G: 0xcd, B: 0x8a, A: 0xff},
		color.RGBA{R: 0x0, G: 0xcf, B: 0x8f, A: 0xff},
		color.RGBA{R: 0x0, G: 0xd1, B: 0x93, A: 0xff},
		color.RGBA{R: 0x0, G: 0xd3, B: 0x97, A: 0xff},
		color.RGBA{R: 0x0, G: 0xd5, B: 0x9c, A: 0xff},
		color.RGBA{R: 0x0, G: 0xd7, B: 0xa0, A: 0xff},
		color.RGBA{R: 0x0, G: 0xd8, B: 0xa5, A: 0xff},
		color.RGBA{R: 0x0, G: 0xdb, B: 0xab, A: 0xff},
		color.RGBA{R: 0x0, G: 0xde, B: 0xb2, A: 0xff},
		color.RGBA{R: 0x0, G: 0xe0, B: 0xb8, A: 0xff},
		color.RGBA{R: 0x0, G: 0xe3, B: 0xbe, A: 0xff},
		color.RGBA{R: 0x0, G: 0xe5, B: 0xc5, A: 0xff},
		color.RGBA{R: 0x0, G: 0xe7, B: 0xcb, A: 0xff},
		color.RGBA{R: 0x0, G: 0xe9, B: 0xd1, A: 0xff},
		color.RGBA{R: 0x0, G: 0xea, B: 0xd6, A: 0xff},
		color.RGBA{R: 0x0, G: 0xea, B: 0xdc, A: 0xff},
		color.RGBA{R: 0x0, G: 0xea, B: 0xe1, A: 0xff},
		color.RGBA{R: 0x0, G: 0xea, B: 0xe6, A: 0xff},
		color.RGBA{R: 0x0, G: 0xea, B: 0xea, A: 0xff},
		color.RGBA{R: 0x0, G: 0xea, B: 0xee, A: 0xff},
		color.RGBA{R: 0x0, G: 0xea, B: 0xf2, A: 0xff},
		color.RGBA{R: 0x0, G: 0xea, B: 0xf6, A: 0xff},
		color.RGBA{R: 0x0, G: 0xea, B: 0xf8, A: 0xff},
		color.RGBA{R: 0x0, G: 0xea, B: 0xfb, A: 0xff},
		color.RGBA{R: 0x0, G: 0xea, B: 0xfe, A: 0xff},
		color.RGBA{R: 0x0, G: 0xea, B: 0xff, A: 0xff},
		color.RGBA{R: 0x0, G: 0xe8, B: 0xff, A: 0xff},
		color.RGBA{R: 0x0, G: 0xe4, B: 0xff, A: 0xff},
		color.RGBA{R: 0x0, G: 0xe0, B: 0xff, A: 0xff},
		color.RGBA{R: 0x0, G: 0xdb, B: 0xff, A: 0xff},
		color.RGBA{R: 0x0, G: 0xd6, B: 0xfe, A: 0xff},
		color.RGBA{R: 0x0, G: 0xd0, B: 0xfc, A: 0xff},
		color.RGBA{R: 0x0, G: 0xca, B: 0xfa, A: 0xff},
		color.RGBA{R: 0x0, G: 0xc3, B: 0xf7, A: 0xff},
		color.RGBA{R: 0x0, G: 0xbc, B: 0xf4, A: 0xff},
		color.RGBA{R: 0x0, G: 0xb4, B: 0xf0, A: 0xff},
		color.RGBA{R: 0x0, G: 0xad, B: 0xec, A: 0xff},
		color.RGBA{R: 0x0, G: 0xa4, B: 0xe8, A: 0xff},
		color.RGBA{R: 0x0, G: 0x9c, B: 0xe4, A: 0xff},
		color.RGBA{R: 0x0, G: 0x93, B: 0xde, A: 0xff},
		color.RGBA{R: 0x0, G: 0x8b, B: 0xda, A: 0xff},
		color.RGBA{R: 0x0, G: 0x82, B: 0xd5, A: 0xff},
		color.RGBA{R: 0x0, G: 0x7a, B: 0xd0, A: 0xff},
		color.RGBA{R: 0x0, G: 0x75, B: 0xcd, A: 0xff},
		color.RGBA{R: 0x0, G: 0x70, B: 0xcb, A: 0xff},
		color.RGBA{R: 0x0, G: 0x6b, B: 0xc7, A: 0xff},
		color.RGBA{R: 0x0, G: 0x63, B: 0xc4, A: 0xff},
		color.RGBA{R: 0x0, G: 0x5d, B: 0xc1, A: 0xff},
		color.RGBA{R: 0x0, G: 0x56, B: 0xbd, A: 0xff},
		color.RGBA{R: 0x0, G: 0x4e, B: 0xb8, A: 0xff},
		color.RGBA{R: 0x0, G: 0x47, B: 0xb4, A: 0xff},
		color.RGBA{R: 0x0, G: 0x41, B: 0xaf, A: 0xff},
		color.RGBA{R: 0x0, G: 0x3a, B: 0xab, A: 0xff},
		color.RGBA{R: 0x0, G: 0x34, B: 0xa7, A: 0xff},
		color.RGBA{R: 0x0, G: 0x2e, B: 0xa2, A: 0xff},
		color.RGBA{R: 0x0, G: 0x28, B: 0x9d, A: 0xff},
		color.RGBA{R: 0x0, G: 0x23, B: 0x98, A: 0xff},
		color.RGBA{R: 0x0, G: 0x1e, B: 0x93, A: 0xff},
		color.RGBA{R: 0x0, G: 0x1a, B: 0x8e, A: 0xff},
		color.RGBA{R: 0x0, G: 0x16, B: 0x88, A: 0xff},
		color.RGBA{R: 0x0, G: 0x12, B: 0x83, A: 0xff},
		color.RGBA{R: 0x0, G: 0xf, B: 0x7e, A: 0xff},
		color.RGBA{R: 0x0, G: 0xc, B: 0x78, A: 0xff},
		color.RGBA{R: 0x0, G: 0x9, B: 0x73, A: 0xff},
		color.RGBA{R: 0x1, G: 0x8, B: 0x6e, A: 0xff},
		color.RGBA{R: 0x1, G: 0x6, B: 0x6a, A: 0xff},
		color.RGBA{R: 0x1, G: 0x5, B: 0x65, A: 0xff},
		color.RGBA{R: 0x2, G: 0x4, B: 0x61, A: 0xff},
		color.RGBA{R: 0x3, G: 0x4, B: 0x5c, A: 0xff},
		color.RGBA{R: 0x4, G: 0x5, B: 0x59, A: 0xff},
		color.RGBA{R: 0x5, G: 0x5, B: 0x55, A: 0xff},
		color.RGBA{R: 0x6, G: 0x6, B: 0x52, A: 0xff},
		color.RGBA{R: 0x7, G: 0x7, B: 0x4f, A: 0xff},
		color.RGBA{R: 0x8, G: 0x8, B: 0x4d, A: 0xff},
		color.RGBA{R: 0xa, G: 0xa, B: 0x4d, A: 0xff},
		color.RGBA{R: 0xc, G: 0xc, B: 0x4d, A: 0xff},
		color.RGBA{R: 0xe, G: 0xe, B: 0x4c, A: 0xff},
		color.RGBA{R: 0x10, G: 0x10, B: 0x4a, A: 0xff},
		color.RGBA{R: 0x13, G: 0x13, B: 0x49, A: 0xff},
		color.RGBA{R: 0x15, G: 0x15, B: 0x48, A: 0xff},
		color.RGBA{R: 0x18, G: 0x18, B: 0x47, A: 0xff},
		color.RGBA{R: 0x1a, G: 0x1a, B: 0x45, A: 0xff},
		color.RGBA{R: 0x1d, G: 0x1d, B: 0x46, A: 0xff},
		color.RGBA{R: 0x20, G: 0x20, B: 0x45, A: 0xff},
		color.RGBA{R: 0x23, G: 0x23, B: 0x44, A: 0xff},
		color.RGBA{R: 0x25, G: 0x25, B: 0x43, A: 0xff},
		color.RGBA{R: 0x28, G: 0x28, B: 0x43, A: 0xff},
		color.RGBA{R: 0x2a, G: 0x2a, B: 0x41, A: 0xff},
		color.RGBA{R: 0x2c, G: 0x2c, B: 0x41, A: 0xff},
		color.RGBA{R: 0x2e, G: 0x2e, B: 0x40, A: 0xff},
		color.RGBA{R: 0x30, G: 0x30, B: 0x3f, A: 0xff},
		color.RGBA{R: 0x31, G: 0x32, B: 0x3e, A: 0xff},
		color.RGBA{R: 0x33, G: 0x33, B: 0x3d, A: 0xff},
		color.RGBA{R: 0x35, G: 0x34, B: 0x3d, A: 0xff},
	}
}

type OverlayShape int

const (
	OverlayShapeRectangle OverlayShape = iota
	OverlayShapeDot
)

// A DataPoint to be plotted.
type DataPoint interface {
	X() float64
	Y() float64
	Dx() int
	Dy() int
}

type apoint struct {
	x  float64
	y  float64
	dx int
	dy int
}

func (a apoint) X() float64 {
	return a.x
}

func (a apoint) Y() float64 {
	return a.y
}

func (a apoint) Dx() int {
	return a.dx
}

func (a apoint) Dy() int {
	return a.dy
}

// P is a shorthand simple datapoint constructor.
func P(x, y float64, dx, dy int) DataPoint {
	return apoint{x, y, dx, dy}
}

// Heatmap draws a heatmap.
//
// size is the size of the image to crate
// opacity is the alpha value (0-255) of the impact of the image overlay  (lower is more transparent)
// overlayRgbaVal is the alpha value of the image overlay (higher is more vivid). If nil, it will be auto-computed.
// scheme is the color palette to choose from the overlay
func Heatmap(size image.Rectangle, points []DataPoint, overlayRgbaVal *uint8, opacity uint8,
	scheme []color.Color, shape OverlayShape) image.Image {

	// Draw black/alpha into the image
	bw := image.NewRGBA(size)
	placePoints(shape, bw, points, overlayRgbaVal)

	rv := image.NewRGBA(size)

	// Then we transplant the pixels one at a time pulling from our color map
	warm(rv, bw, opacity, scheme)

	return rv
}

func placePoints(shape OverlayShape, bw *image.RGBA, points []DataPoint, overlayRgbaVal *uint8) {
	for _, p := range points {
		var img draw.Image
		switch shape {
		case OverlayShapeRectangle:
			img = makeRect(p.Dy(), p.Dx(), overlayRgbaVal)
		case OverlayShapeDot:
			img = makeDot(math.Min(float64(p.Dx()), float64(p.Dy())), overlayRgbaVal)
		}

		draw.Draw(bw, image.Rect(int(p.X()), int(p.Y()), int(p.X())+p.Dx(), int(p.Y())+p.Dy()), img,
			image.Point{}, draw.Over)
	}
}

func warm(out, in draw.Image, opacity uint8, colors []color.Color) {
	draw.Draw(out, out.Bounds(), image.Transparent, image.Point{}, draw.Src)
	bounds := in.Bounds()
	collen := float64(len(colors))
	wg := &sync.WaitGroup{}

	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		wg.Add(1)
		go func(x int) {
			defer wg.Done()

			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				col := in.At(x, y)
				_, _, _, alpha := col.RGBA()

				if alpha > 0 {
					percent := float64(alpha) / float64(0xffff)
					template := colors[int((collen-1)*(1.0-percent))]
					tr, tg, tb, ta := template.RGBA()
					ta /= 256
					outalpha := uint8(float64(ta) *
						(float64(opacity) / 256.0))
					outcol := color.NRGBA{
						uint8(tr / 256),
						uint8(tg / 256),
						uint8(tb / 256),
						uint8(outalpha)}
					out.Set(x, y, outcol)
				}
			}
		}(x)
	}
	wg.Wait()
}

func makeRect(height, width int, rgbaVal *uint8) draw.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			var rgbVal uint8
			if rgbaVal != nil {
				rgbVal = *rgbaVal
			} else {
				rgbVal = uint8(128)
			}
			rgba := color.NRGBA{0, 0, 0, rgbVal}
			img.Set(int(x), int(y), rgba)
		}
	}

	return img
}

func makeDot(size float64, rgbaVal *uint8) draw.Image {
	i := image.NewRGBA(image.Rect(0, 0, int(size), int(size)))

	md := 0.5 * math.Sqrt(math.Pow(float64(size)/2.0, 2)+math.Pow((float64(size)/2.0), 2))
	for x := float64(0); x < size; x++ {
		for y := float64(0); y < size; y++ {
			d := math.Sqrt(math.Pow(x-size/2.0, 2) + math.Pow(y-size/2.0, 2))
			if d < md {
				var rgbVal uint8
				if rgbaVal != nil {
					rgbVal = *rgbaVal
				} else {
					rgbVal = uint8(200.0*d/md + 50.0)
				}
				rgba := color.NRGBA{0, 0, 0, 255 - rgbVal}
				i.Set(int(x), int(y), rgba)
			}
		}
	}

	return i
}