/*
 * File: error.go
 * Project: common
 * File Created: Sunday, 21st August 2022 12:34:15 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package common

import (
	"fmt"
)

func CombineErrors(errors []error) error {
	var err error
	for _, e := range errors {
		if e != nil {
			if err == nil {
				err = fmt.Errorf("%w", e)
			} else {
				err = fmt.Errorf("%s %w", err.Error(), e)
			}
		}
	}
	return err
}
