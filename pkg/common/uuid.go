/*
 * File: uuid.go
 * Project: common
 * File Created: Tuesday, 16th August 2022 12:50:24 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package common

import "github.com/google/uuid"

func ShortUUID(l int) string {
	s := uuid.New().String()
	return s[len(s)-l:]
}
