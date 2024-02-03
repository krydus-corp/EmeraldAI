/*
 * File: upload.go
 * Project: aws
 * File Created: Friday, 19th March 2021 6:10:53 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package awss3

import (
	"context"
	"io"
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

// S3Uploader
type S3Uploader struct {
	*manager.Uploader
}

func NewS3Uploader(partsize int64, concurrency int, bucket string) (*S3Uploader, error) {
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

	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = partsize       // The minimum/default allowed part size is 5MB
		u.Concurrency = concurrency // default is 5
	})

	return &S3Uploader{uploader}, nil
}

func (u *S3Uploader) Upload(reader io.Reader, bucket, key string) (string, error) {

	// Upload the file to S3.
	result, err := u.Uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   reader,
		ACL:    types.ObjectCannedACLPrivate,
	})

	if err != nil {
		return "", errors.Wrap(err, "failed to upload file")
	}

	return result.Location, nil
}
