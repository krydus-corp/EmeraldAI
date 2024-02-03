/*
 * File: tar.go
 * Project: content
 * File Created: Friday, 26th March 2021 4:15:06 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package archive

import (
	"fmt"
	"path"
	"strings"
)

// Unarchiver is a type that can extract archive files.
type Unarchiver interface {
	Unarchive(source []byte) ([]ArchiveResult, error)
}

type ArchiveResult struct {
	FileBytes []byte
	FileName  string
}

func Unarchive(filename string, filebytes []byte) (files []ArchiveResult, err error) {
	ext := strings.ToLower(path.Ext(filename))

	switch ext {
	case ".tar":
		files, err = Untar(filebytes, false)
		if err != nil {
			return nil, fmt.Errorf("unable to unarchive tarfile; file=%s, err=%s", filename, err.Error())
		}
	case ".gz", ".gzip":
		files, err = Untar(filebytes, true)
		if err != nil {
			return nil, fmt.Errorf("unable to unarchive tarfile; file=%s, err=%s", filename, err.Error())
		}
	case ".zip":
		files, err = Unzip(filebytes)
		if err != nil {
			return nil, fmt.Errorf("unable to unarchive zipfile; file=%s, err=%s", filename, err.Error())
		}
	default:
		return nil, fmt.Errorf("unsupported filetype=%s", ext)
	}

	return
}
