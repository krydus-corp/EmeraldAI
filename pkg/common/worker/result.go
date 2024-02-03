/*
 * File: result.go
 * Project: worker
 * File Created: Sunday, 11th September 2022 3:50:43 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package worker

type Result[T any] struct {
	Value T
	Err   error
}
