/*
 * File: bucket.go
 * Project: aws
 * File Created: Friday, 19th March 2021 6:24:02 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package awss3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/pkg/errors"
)

// Custom errors
var (
	ErrBucketAlreadyExists = fmt.Errorf("bucket already exists")
	ErrBucketAlreadyOwned  = fmt.Errorf("bucket already owned")
	ErrFileDoesNotExist    = fmt.Errorf("file does not exist")
)

// CreateBucket creates an S3 bucket.
func CreateBucket(name string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}

	svc := s3.NewFromConfig(cfg)
	input := &s3.CreateBucketInput{
		Bucket: aws.String(name),
		ACL:    types.BucketCannedACLPrivate,
	}

	if _, err := svc.CreateBucket(context.TODO(), input); err != nil {
		var (
			bae *types.BucketAlreadyExists
			bao *types.BucketAlreadyOwnedByYou
		)
		if errors.As(err, &bae) {
			return ErrBucketAlreadyExists
		}
		if errors.As(err, &bao) {
			return ErrBucketAlreadyOwned
		}
		return errors.Wrap(err, "Unexpected error during bucket creation")
	}

	corsConfig := types.CORSConfiguration{
		CORSRules: []types.CORSRule{
			{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "HEAD", "PUT"},
				AllowedHeaders: []string{"*"},
				ExposeHeaders:  []string{},
			},
		},
	}
	params := s3.PutBucketCorsInput{
		Bucket:            aws.String(name),
		CORSConfiguration: &corsConfig,
	}

	_, err = svc.PutBucketCors(context.TODO(), &params)
	if err != nil {
		return errors.Wrap(err, "Unexpected error while setting CORSRule on bucket")
	}

	return nil
}
