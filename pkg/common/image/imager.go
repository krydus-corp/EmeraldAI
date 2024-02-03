// Package image provides image processing utilities
/*
 * File: imager.go
 * Project: image
 * File Created: Saturday, 4th April 2020 7:16:14 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package image

import (
	"fmt"
	"sync"

	guuid "github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// Imager is a struct for holding an Imager's context variables
type Imager struct {
	// TypeConversion is the image MIME type to convert
	TypeConversion *string
	// SizeConversion is the resize parameters to use when resizing images
	SizeConversion *SizeConversionParams

	// Concurrency of this Imager
	Concurrency int

	// Sync vars
	InChan   chan *Image
	OutChan  chan *Image
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// SizeConversionParams is a struct containing the image resize params
type SizeConversionParams struct {
	Height int
	Width  int
}

// Image is a struct for representing an image to be processed
type Image struct {
	OriginalFilepath  string
	ProcessedFilepath string
	ImageBytes        []byte
	Err               error
}

// NewImager is a function for initializing a new Imager object
func NewImager(concurrency int, typeConv *string, sizeConv *SizeConversionParams) (*Imager, error) {
	if typeConv != nil {
		if !checkSupportedContentType(*typeConv) {
			return nil, fmt.Errorf("unsupported type conversion '%s'", *typeConv)
		}
	}

	return &Imager{
		TypeConversion: typeConv,
		SizeConversion: sizeConv,
		Concurrency:    concurrency,
		InChan:         make(chan *Image, 128),
		OutChan:        make(chan *Image, 128),
		stopChan:       make(chan struct{}, 1),
	}, nil
}

// Start kicks off the Imager worker routines
func (imgr *Imager) Start() error {

	// Start a fixed number of goroutines to read and digest images.
	imgr.wg.Add(imgr.Concurrency)
	for i := 0; i < imgr.Concurrency; i++ {
		go func() {
			imgr.work()
			imgr.wg.Done()
		}()
	}

	return nil
}

// Stop the Imager worker routines and wait for graceful exit
func (imgr *Imager) Stop() {
	close(imgr.stopChan)
	imgr.wg.Wait()
}

func (imgr *Imager) work() {
	id := guuid.New()
	log.Debugf("Starting Imager worker routine [%s]\n", id.String())
	defer log.Debugf("Exiting Imager worker routine [%s]\n", id.String())

	for {
		select {
		case img := <-imgr.InChan:
			img.ProcessedFilepath = img.OriginalFilepath

			if imgr.TypeConversion != nil {
				imgC, err := ConvertImgType(img.ImageBytes, *imgr.TypeConversion)
				if err != nil {
					img.Err = err
				} else {
					img.ImageBytes = imgC
					img.ProcessedFilepath = updatePathExtension(img.OriginalFilepath, *imgr.TypeConversion)
				}
			}

			if imgr.SizeConversion != nil {
				imgR, err := ResizeImage(img.ImageBytes, imgr.SizeConversion.Width, imgr.SizeConversion.Height)
				if err != nil {
					img.Err = err
				} else {
					img.ImageBytes = imgR
				}
			}

			imgr.OutChan <- img

		case <-imgr.stopChan:
			return
		}
	}
}
