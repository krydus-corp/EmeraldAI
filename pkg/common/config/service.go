/*
 * File: service.go
 * Project: config
 * File Created: Friday, 6th January 2023 9:54:01 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package config

import (
	"net/url"
	"os"
	"strings"

	s3 "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/aws/s3"
)

// NewConfiguration reads a configuration file and returns the service's configuration obj
func NewConfiguration[T any](configPath string, s3Path bool) (*T, error) {
	var config T

	if s3Path {
		u, err := url.Parse(configPath)
		if err != nil {
			return nil, err
		}

		bucket := u.Host
		key := strings.TrimLeft(u.Path, "/")

		fileBytes, err := s3.NewS3().GetObject(bucket, key)
		if err != nil {
			return nil, err
		}

		file, err := os.CreateTemp("", "emld.*.yml")
		if err != nil {
			return nil, err
		}
		defer os.Remove(file.Name())

		_, err = file.Write(fileBytes)
		if err != nil {
			return nil, err
		}

		err = ReadConfig(file.Name(), &config)
		if err != nil {
			return nil, err
		}

	} else {
		err := ReadConfig(configPath, &config)
		if err != nil {
			return nil, err
		}
	}

	return &config, nil
}
