/*
 * File: zipwriter_test.go
 * Project: archive
 * File Created: Monday, 9th January 2023 11:16:31 am
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package archive

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"testing"

	common "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	s3 "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/aws/s3"
)

const (
	LogLvl              = "info"
	Const_S3Concurrency = 10
	Const_S3Partsize    = 1024 * 1024 * 5 // 5 MB
	Bucket              = "emld-user-data-integration-test"
	Archive             = "archive-test"
	Filepath            = "/Users/anonymous /Desktop/military-dataset-small"
	FilesToArchive      = 1000
)

func TestZipWriterIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Initialize an S3 downloader
	s3downloader, err := s3.NewS3Downloader(Const_S3Partsize, Const_S3Concurrency, Bucket)
	if err != nil {
		panic("S3 downloader initialization error; " + err.Error())
	}

	// Initialize an S3 uploader
	s3uploader, err := s3.NewS3Uploader(Const_S3Partsize, Const_S3Concurrency, Bucket)
	if err != nil {
		panic("S3 uploader initialization error; " + err.Error())
	}

	zipw := New(
		s3downloader.Downloader,
		s3uploader.Uploader,
	)

	// Upload archive
	inChan := make(chan *ObjectInput, 10)
	doneChan := make(chan error, 1)

	go func() {
		err := zipw.ZipS3Files(inChan, &ObjectOutput{Bucket: common.Ptr(Bucket), Key: common.Ptr(fmt.Sprintf("%s-%s.zip", Archive, common.ShortUUID(4)))})
		doneChan <- err
	}()

	// Create an 100 x 50 image
	img := image.NewRGBA(image.Rect(0, 0, 100, 50))
	img.Set(2, 3, color.RGBA{255, 0, 0, 255})

	// Save to out.png
	b := bytes.Buffer{}
	png.Encode(&b, img)

	log.Printf("Starting feed loop...")
	for i := 0; i < FilesToArchive; i++ {
		img := fmt.Sprintf("test-image-%s.png", common.ShortUUID(6))
		imgMap := map[string][]byte{img: b.Bytes()}
		inChan <- &ObjectInput{Bucket: nil, Key: nil, RawBytes: imgMap}
	}
	log.Printf("Ending feed loop...")

	close(inChan)

	err = <-doneChan
	if err != nil {
		panic(err)
	}

	log.Printf("Exiting...")

}
