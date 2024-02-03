/*
 * File: s3.go
 * Project: aws
 * File Created: Monday, 15th March 2021 10:09:45 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package awss3

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/ratelimit"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/pkg/errors"
)

const (
	poolSize = 1024 * 1024 * 5
)

// S3Downloader ...
type S3Downloader struct {
	Client     *s3.Client
	Downloader *manager.Downloader
}

// NewS3Downloader ...
func NewS3Downloader(partsize int64, concurrency int, bucket string) (*S3Downloader, error) {
	bucketAcceleration := types.BucketAccelerateStatusSuspended
	useAcc, _ := strconv.ParseBool(os.Getenv("S3_USE_ACCELERATE_ENDPOINT"))
	if useAcc {
		bucketAcceleration = types.BucketAccelerateStatusEnabled
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRetryer(func() aws.Retryer {
		return retry.NewStandard(func(so *retry.StandardOptions) {
			so.RateLimiter = ratelimit.NewTokenRateLimit(1000000)
			so.Backoff = retry.NewExponentialJitterBackoff(60 * time.Second)
			so.MaxAttempts = 3
		})
	}))
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg)
	if _, err := client.PutBucketAccelerateConfiguration(context.TODO(), &s3.PutBucketAccelerateConfigurationInput{
		AccelerateConfiguration: &types.AccelerateConfiguration{Status: bucketAcceleration},
		Bucket:                  &bucket}); err != nil {
		return nil, err
	}

	s3Downloader := manager.NewDownloader(client, func(d *manager.Downloader) {
		d.BufferProvider = manager.NewPooledBufferedWriterReadFromProvider(poolSize)
		d.PartSize = partsize       // The minimum/default allowed part size is 5MB
		d.Concurrency = concurrency // default is 5
	})

	return &S3Downloader{Client: client, Downloader: s3Downloader}, nil
}

// Download ...
func (d *S3Downloader) Download(bucket, key string) ([]byte, int, error) {
	res, err := d.Client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return nil, http.StatusNotFound, err

		}
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to retrieve file header details for bucket=%s key=%s, %s", bucket, key, err.Error())
	}
	buff := manager.NewWriteAtBuffer(make([]byte, int(res.ContentLength)))

	if _, err := d.Downloader.Download(context.TODO(), buff, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to download file, %s", err.Error())
	}

	return buff.Bytes(), 200, nil
}
