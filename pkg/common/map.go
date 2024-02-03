/*
 * File: map.go
 * Project: common
 * File Created: Tuesday, 16th November 2021 9:53:29 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package common

func ReverseMapStringInt(m map[string]int) map[int]string {
	n := make(map[int]string, len(m))
	for k, v := range m {
		n[v] = k
	}
	return n
}

func MapStringStructToSlice(m map[string]struct{}) (slice []string) {
	for k := range m {
		slice = append(slice, k)
	}
	return
}
