/*
 * File: file.go
 * Project: common
 * File Created: Sunday, 11th September 2022 1:28:40 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package common

import (
	"io"
	"mime/multipart"
	"os"
)

func ReadFile(file *multipart.FileHeader) ([]byte, error) {

	if file != nil {
		if file.Size > 0 {
			f, err := file.Open()
			if err != nil {
				return nil, err
			}
			return io.ReadAll(f)
		}
	}

	return nil, nil
}

func FileSize(file *multipart.FileHeader) (int64, error) {
	var fileSize int64 = 0
	src, err := file.Open()
	if err != nil {
		return 0, err
	}
	defer func() {
		src.Seek(0, 0)
		src.Close()
	}()

	switch t := src.(type) {
	case *os.File:
		fi, err := t.Stat()
		if err != nil {
			return 0, err
		}
		fileSize = fi.Size()
	default:
		var sr int64
		sr, err := src.Seek(0, 2) // file size in first 2 bytes
		if err != nil {
			return 0, err
		}
		_, err = src.Seek(0, 0) // reset
		if err != nil {
			return 0, err
		}
		fileSize = sr
	}
	return fileSize, nil
}
