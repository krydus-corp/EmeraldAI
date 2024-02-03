/*
 * File: type.go
 * Project: common
 * File Created: Tuesday, 27th September 2022 1:54:14 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package common

func Ptr[T any](t T) *T {
	return &t
}
