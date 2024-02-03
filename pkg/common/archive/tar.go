/*
 * File: tar.go
 * Project: archive
 * File Created: Sunday, 24th October 2021 3:10:37 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
)

func Untar(source []byte, isGzip bool) ([]ArchiveResult, error) {
	reader := bytes.NewReader(source)
	var tarReader *tar.Reader

	if isGzip {
		gzipReader, err := gzip.NewReader(reader)
		if err != nil {
			return nil, err
		}
		defer gzipReader.Close()

		tarReader = tar.NewReader(gzipReader)
	} else {
		tarReader = tar.NewReader(reader)
	}

	files := []ArchiveResult{}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		filename := header.Name
		info := header.FileInfo()
		if info.IsDir() {
			continue
		}

		var buf bytes.Buffer

		_, err = io.Copy(&buf, tarReader)
		if err != nil {
			return nil, err
		}

		files = append(files, ArchiveResult{FileBytes: buf.Bytes(), FileName: filename})
	}

	return files, nil
}
