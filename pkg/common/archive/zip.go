/*
 * File: zip.go
 * Project: archive
 * File Created: Sunday, 24th October 2021 3:11:05 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package archive

import (
	"archive/zip"
	"bytes"
	"io"
)

func Unzip(source []byte) ([]ArchiveResult, error) {
	reader := bytes.NewReader(source)
	zipReader, err := zip.NewReader(reader, reader.Size())
	if err != nil {
		return nil, err
	}

	files := []ArchiveResult{}

	for _, f := range zipReader.File {
		filename := f.FileHeader.Name
		info := f.FileHeader.FileInfo()
		if info.IsDir() {
			continue
		}

		var buf bytes.Buffer

		rc, err := f.Open()
		if err != nil {
			return nil, err
		}

		_, err = io.Copy(&buf, rc)
		if err != nil {
			return nil, err
		}

		rc.Close()

		files = append(files, ArchiveResult{FileBytes: buf.Bytes(), FileName: filename})
	}

	return files, nil
}
