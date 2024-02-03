package models

import (
	"encoding/json"
	"errors"
)

var (
	ErrInvalidResizeMode = errors.New("invalid preprocessor 'resize'; mode must be one of stretch, fit, fill")
)

// Model represents Preprocessors domain model
//
// swagger:model Preprocessors
type Preprocessors struct {
	Resize      *Resize      `json:"resize,omitempty" bson:"resize,omitempty"`
	Greyscale   *Greyscale   `json:"greyscale,omitempty" bson:"greyscale,omitempty"`
	PercentCrop *PercentCrop `json:"percent_crop,omitempty" bson:"percent_crop,omitempty"`
}

func (p *Preprocessors) ToJSON() ([]byte, error) {
	return json.MarshalIndent(p, "", " ")
}

// Model represents Augmentations domain model
//
// swagger:model Augmentations
type Augmentations struct {
	Flip            *Flip            `json:"flip,omitempty" bson:"flip,omitempty"`
	Rotate          *Rotate          `json:"rotate,omitempty" bson:"rotate,omitempty"`
	ColorJitter     *ColorJitter     `json:"color_jitter,omitempty" bson:"color_jitter,omitempty"`
	RandomCrop      *RandomCrop      `json:"random_crop,omitempty" bson:"random_crop,omitempty"`
	GaussianBlur    *GaussianBlur    `json:"gaussian_blur,omitempty" bson:"guassian_blur,omitempty"`
	Grey            *Grey            `json:"greyscale,omitempty" bson:"greyscale,omitempty"`
	RandomErasing   *RandomErasing   `json:"random_erasing,omitempty" bson:"random_erasing,omitempty"`
	Noise           *Noise           `json:"noise,omitempty" bson:"noise,omitempty"`
	JpegCompression *JpegCompression `json:"jpeg_compression,omitempty" bson:"jpeg_compression,omitempty"`
}

func (a *Augmentations) ToJSON() ([]byte, error) {
	return json.MarshalIndent(a, "", " ")
}

// Preprocessing is a contract for new preprocessors & augmentations
type Preprocessing interface {
	Valid() error
}

// Resize images based on set mode
type Resize struct {
	// example: stretch || fill || fit
	Mode string `json:"mode" bson:"mode" validate:"required,oneof=stretch fill fit"`
	Size struct {
		// example: 720
		Height int `json:"height" bson:"height" validate:"required,numeric,min=0"`
		// example: 1080
		Width int `json:"width" bson:"width" validate:"required,numeric,min=0"`
	} `json:"size" bson:"size" validate:"required"`
}

func (r *Resize) Valid() error {
	if r.Mode != "stretch" && r.Mode != "fill" && r.Mode != "fit" {
		return ErrInvalidResizeMode
	}
	return nil
}

// Convert images to greyscale using color channel
type Greyscale struct {
	// example: UNUSED RESERVED
	Channels string `json:"channel" bson:"channel" validate:"required"`
}

func (g *Greyscale) Valid() error {
	return nil
}

// Crop images by percent
type PercentCrop struct {
	// example: 0.5
	// min: 0
	// max: 1
	Height float64 `json:"height" bson:"height" validate:"required,numeric,min=0,max=1"`
	// example: 0.5
	// min: 0
	// max: 1
	Width float64 `json:"width" bson:"width" validate:"required,numeric,min=0,max=1"`
}

func (p *PercentCrop) Valid() error {
	return nil
}

// Flip images
type Flip struct {
	// example: true
	Horizontal bool `json:"horizontal" bson:"horizontal"`
	// example: true
	Vertical bool `json:"vertical" bson:"vertical"`
}

func (f *Flip) Valid() error {
	return nil
}

// Rotate images by degree
type Rotate struct {
	// Degree of rotation
	// example: 90
	// min: 0
	// max: 360
	Degree int `json:"degree" bson:"degree" validate:"required"`
}

func (r *Rotate) Valid() error {
	return nil
}

// Randomly change brightness, contrast, saturation and hue of the images
type ColorJitter struct {
	// example: 0.5
	Brightness float32 `json:"brightness" bson:"brightness" validate:"required,numeric,min=0.0,max=1.0"`
	// example: 0.5
	Contrast float32 `json:"contrast" bson:"contrast" validate:"required,numeric,min=0.0,max=1.0"`
	// example: 0.5
	Saturation float32 `json:"saturation" bson:"saturation" validate:"required,numeric,min=0.0,max=1.0"`
	// example: 0.5
	Hue float32 `json:"hue" bson:"hue" validate:"required,numeric,min=0.0,max=1.0"`
}

func (c *ColorJitter) Valid() error {
	return nil
}

// Crop images randomly
type RandomCrop struct {
	// example: true
	Enabled bool `json:"enabled" bson:"enabled" validate:"required"`
}

func (r *RandomCrop) Valid() error {
	return nil
}

// Apply gaussian_blur to the images
type GaussianBlur struct {
	// example: 0.5
	Intensity float32 `json:"intensity" bson:"intensity" validate:"required,numeric,min=0.0,max=1.0"`
}

func (r *GaussianBlur) Valid() error {
	return nil
}

// Convert the images to greyscale
type Grey struct {
	// example: true
	Enabled bool `json:"enabled" bson:"enabled" validate:"required"`
}

func (r *Grey) Valid() error {
	return nil
}

// Randomly select rectangles in the images and erase the rectangles pixels
type RandomErasing struct {
	// example: true
	Enabled bool `json:"enabled" bson:"enabled" validate:"required"`
}

func (r *RandomErasing) Valid() error {
	return nil
}

// Add noise to the images
type Noise struct {
	// example: 0.5
	Intensity float32 `json:"intensity" bson:"intensity" validate:"required,numeric,min=0.0,max=1.0"`
}

func (r *Noise) Valid() error {
	return nil
}

// Compress the images
type JpegCompression struct {
	// example: true
	Enabled bool `json:"enabled" bson:"enabled" validate:"required"`
}

func (r *JpegCompression) Valid() error {
	return nil
}
