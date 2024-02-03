/*
 * File: config.go
 * Project: exporter
 * File Created: Thursday, 5th January 2023 8:35:36 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package exporter

import (
	blob "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/blob"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

type Configuration struct {
	DB        *platform.Configuration `yaml:"database,omitempty"`
	BlobStore *blob.Configuration     `yaml:"blob_store,omitempty"`
}
