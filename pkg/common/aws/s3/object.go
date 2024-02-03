/*
 * File: delete.go
 * Project: aws
 * File Created: Friday, 26th March 2021 5:57:09 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package awss3

import (
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3 struct {
	*s3.S3
}

func NewS3() *S3 {
	// The session the S3 Uploader will use
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := s3.New(sess)

	return &S3{svc}
}

func (s *S3) PutObject(bucket, key string, data io.ReadSeeker) error {
	_, err := s.S3.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   data,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *S3) GetObject(bucket, key string) ([]byte, error) {
	out, err := s.S3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	fileBytes, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, err
	}

	return fileBytes, nil
}

func (s *S3) DeleteObject(bucket, key string) error {

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	out, err := s.S3.ListObjects(&s3.ListObjectsInput{MaxKeys: aws.Int64(1), Bucket: &bucket, Prefix: &key})
	if err != nil {
		return err
	}

	// This is a directory
	if len(out.Contents) == 1 {
		err = s.DeleteAllObjects(bucket, key)
	} else {
		// Single object
		_, err = s.S3.DeleteObject(input)
	}

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

func (s *S3) ListObjects(bucket string, prefix string) ([]*s3.Object, *string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	results, err := s.S3.ListObjectsV2(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket:
				return []*s3.Object{}, nil, nil
			default:
				return []*s3.Object{}, nil, err
			}
		} else {
			return []*s3.Object{}, nil, err
		}
	}

	return results.Contents, results.ContinuationToken, nil
}

func (s *S3) DeleteAllObjects(bucket, prefix string) error {

	iter := s3manager.NewDeleteListIterator(s.S3, &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	if err := s3manager.NewBatchDeleteWithClient(s.S3).Delete(aws.BackgroundContext(), iter); err != nil {
		return fmt.Errorf("unable to delete objects from bucket %q/%q, %s", bucket, prefix, err.Error())
	}

	return nil
}

func (s *S3) ObjectExists(bucket string, key string) (bool, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(key),
	}

	results, err := s.S3.ListObjectsV2(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket:
				return false, nil
			default:
				return false, err
			}
		} else {
			return false, err
		}
	}

	if *results.KeyCount > 0 {
		return true, nil
	}

	return false, nil
}
