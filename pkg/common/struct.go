/*
 * File: struct.go
 * Project: common
 * File Created: Thursday, 2nd December 2021 9:22:33 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package common

import (
	"fmt"
	"reflect"
)

func SelectStructFields(_struct interface{}, tag string) (map[string]interface{}, error) {
	rt, rv := reflect.TypeOf(_struct), reflect.ValueOf(_struct)
	if rt.Kind() != reflect.Struct {
		return make(map[string]interface{}, 0), fmt.Errorf("expected struct type")
	}

	out := make(map[string]interface{}, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		key := field.Tag.Get(tag)
		if key != "" {
			out[key] = rv.Field(i).Interface()
		}
	}
	return out, nil
}

func GetStructTag(f reflect.StructField, tagName string) string {
	return string(f.Tag.Get(tagName))
}
