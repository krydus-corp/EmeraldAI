/*
 * File: error.go
 * Project: train
 * File Created: Tuesday, 16th August 2022 6:36:37 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package train

import "errors"

var (
	ErrNoContent    = errors.New("no content to process; training and validation annotations should be > 1")
	ErrInvalidState = errors.New("invalid model state")
)
