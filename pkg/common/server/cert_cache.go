/*
 * File: cert_cache.go
 * Project: server
 * File Created: Thursday, 24th March 2022 6:31:38 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package server

import (
	"bytes"
	"context"
	"log"
	"path"

	"golang.org/x/crypto/acme/autocert"

	s3 "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/aws/s3"
)

// S3Cache implements Cache using a bucket on S3.
// If the bucket does not exist, it will be created it.
type S3Cache struct {
	Bucket string
	Prefix string
}

// Get reads a certificate data from the specified file name.
func (s S3Cache) Get(ctx context.Context, key string) ([]byte, error) {
	var (
		data []byte
		err  error
		done = make(chan struct{})
	)

	log.Printf("Retreiving cert from cache -> s3://%s", path.Join(s.Bucket, s.Prefix, key))

	go func() {
		data, err = s3.NewS3().GetObject(s.Bucket, path.Join(s.Prefix, key))
		close(done)
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
	}
	if err != nil {
		return nil, autocert.ErrCacheMiss
	}

	log.Printf("Retrieved cert from cache -> s3://%s", path.Join(s.Bucket, s.Prefix, key))
	return data, err
}

// Put writes the certificate data to the specified key.
func (s S3Cache) Put(ctx context.Context, key string, data []byte) error {
	var (
		err  error
		done = make(chan struct{})
	)

	log.Printf("Putting cert to cache -> s3://%s", path.Join(s.Bucket, s.Prefix, key))

	go func() {
		err = s3.NewS3().PutObject(s.Bucket, path.Join(s.Prefix, key), bytes.NewReader((data)))
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
	}
	if err != nil {
		return err
	}
	return nil
}

// Delete removes the specified key.
func (s S3Cache) Delete(ctx context.Context, key string) error {
	var (
		err  error
		done = make(chan struct{})
	)

	log.Printf("Deleting cert from cache -> s3://%s", path.Join(s.Bucket, s.Prefix, key))

	go func() {
		err = s3.NewS3().DeleteObject(s.Bucket, path.Join(s.Prefix, key))
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
	}
	if err != nil {
		return err
	}
	return nil
}
