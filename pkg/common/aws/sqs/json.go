/*
 * File: json.go
 * Project: sqs
 * File Created: Friday, 12th November 2021 2:39:58 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package awssqs

type JSONString string

func (js JSONString) MarshalJSON() ([]byte, error) {
	return []byte(js), nil
}
