/*
 * File: slice.go
 * Project: strings
 * File Created: Tuesday, 16th November 2021 3:47:26 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package common

import (
	"sort"
	"strings"

	"golang.org/x/exp/constraints"
)

func StringSliceToLower(strSlice []string) []string {
	list := []string{}
	for _, item := range strSlice {
		list = append(list, strings.ToLower(item))
	}
	return list
}

func RemoveDuplicateStr[T comparable](slice []T) []T {
	allKeys := make(map[T]bool)
	list := []T{}
	for _, item := range slice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func SliceContains[T comparable](slice []T, elem T) bool {
	for _, item := range slice {
		if item == elem {
			return true
		}
	}
	return false
}

func RemoveFromSlice[T comparable](slice []T, elem T) []T {
	newSlice := []T{}

	for _, e := range slice {
		if e != elem {
			newSlice = append(newSlice, e)
		}
	}
	return newSlice
}

func SortSlice[T constraints.Ordered](s []T) {
	sort.Slice(s, func(i, j int) bool {
		return s[i] < s[j]
	})
}
