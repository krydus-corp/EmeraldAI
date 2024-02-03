/*
 * File: blob.go
 * Project: blob
 * File Created: Monday, 15th March 2021 9:32:41 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package blob

import (
	"log"
	"net/http"
	"runtime"
	"sync"

	"github.com/labstack/echo/v4"

	s3 "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/aws/s3"
	worker "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/worker"
)

// Custom errors
var (
	ErrBucketAlreadyExists = echo.NewHTTPError(http.StatusConflict, "Bucket already exists.")
	ErrBucketAlreadyOwned  = echo.NewHTTPError(http.StatusConflict, "Bucket already owned.")
)

// Blob represents the client for the blob store
type Blob struct {
	Uploader   *s3.S3Uploader
	Downloader *s3.S3Downloader
	S3Client   *s3.S3

	DeleteChan chan string

	Bucket string

	stopChan chan struct{}
	wg       sync.WaitGroup
}

type Configuration struct {
	Bucket                  string `mapstructure:"bucket,omitempty" yaml:"bucket,omitempty"`
	Concurrency             int    `mapstructure:"concurrency,omitempty" yaml:"concurrency,omitempty"`
	PartSize                int64  `mapstructure:"partsize,omitempty" yaml:"partsize,omitempty"`
	CreateBucketIfNotExists bool   `mapstructure:"create_bucket_if_not_exists,omitempty" yaml:"create_bucket_if_not_exists,omitempty"`
}

type BlobStore interface {
	Delete(string, string) error
	Get(string, string) ([]byte, error)

	GC()
}

func NewBlob(cfg *Configuration) (*Blob, error) {

	uploader, err := s3.NewS3Uploader(cfg.PartSize, cfg.Concurrency, cfg.Bucket)
	if err != nil {
		return nil, err
	}
	downloader, err := s3.NewS3Downloader(cfg.PartSize, cfg.Concurrency, cfg.Bucket)
	if err != nil {
		return nil, err
	}

	if cfg.CreateBucketIfNotExists {
		err := s3.CreateBucket(cfg.Bucket)
		switch err {
		case nil:
			// pass
		case s3.ErrBucketAlreadyExists:
			return nil, ErrBucketAlreadyExists
		case s3.ErrBucketAlreadyOwned:
			return nil, ErrBucketAlreadyOwned
		default:
			return nil, err
		}
	}

	return &Blob{
		Uploader:   uploader,
		Downloader: downloader,
		Bucket:     cfg.Bucket,
		S3Client:   s3.NewS3(),
		DeleteChan: make(chan string, 1024),
		stopChan:   make(chan struct{}, 1),
		wg:         sync.WaitGroup{},
	}, nil
}

func (b *Blob) Get(bucket, key string) ([]byte, error) {
	content, _, err := b.Downloader.Download(bucket, key)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (b *Blob) GC() {
	pool := worker.New[struct{}](worker.Config{
		Concurrency:   runtime.NumCPU(),
		RetryAttempts: 0,
		Name:          "Blob Garbage Collector",
	}).Start()

	go func() {
		for {
			result := <-pool.OutChan
			if result.Err != nil {
				log.Printf("unexpected GC error; err=%s", result.Err.Error())
			}
		}
	}()

	// Feed the pool
	log.Printf("starting blob GC feed loop")
	defer log.Printf("exiting blob garbage collector")

	b.wg.Add(1)
	defer b.wg.Done()
	defer pool.Stop()

	for {
		select {
		case <-b.stopChan:
			return
		case elem := <-b.DeleteChan:
			log.Printf("deleting object %s/%s", b.Bucket, elem)
			args := deleteArgs{
				s3Client: b.S3Client,
				bucket:   b.Bucket,
				key:      elem,
			}
			pool.InChan <- args.delete
		}
	}
}

type deleteArgs struct {
	s3Client *s3.S3
	bucket   string
	key      string
}

func (a *deleteArgs) delete() (struct{}, error) {
	if err := a.s3Client.DeleteObject(a.bucket, a.key); err != nil {
		log.Printf("error deleting S3 object; bucket=%s, object=%s, err=%s", a.bucket, a.key, err.Error())
		return struct{}{}, err
	}

	return struct{}{}, nil
}

func (b *Blob) Exit() {
	close(b.stopChan)
	b.wg.Wait()
}
