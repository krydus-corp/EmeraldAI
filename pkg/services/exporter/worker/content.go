/*
 * File: content.go
 * Project: export
 * File Created: Tuesday, 30th November 2021 8:08:01 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package worker

import (
	"encoding/json"
)

// Content information
type ContentSlice []Content
type Content struct {
	S3Bucket string `json:"-"`
	S3Key    string `json:"-"`

	Archive          string `json:"archive"`
	Filename         string `json:"filename"`
	FilenameOriginal string `json:"filename_original"`
	MimeType         string `json:"mimetype"`
	Size             int    `json:"size"`
}

func (c ContentSlice) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

type UploadPackage struct {
	ProjectPack []Content
}
