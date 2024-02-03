/*
 * File: env.go
 * Project: runtime
 * File Created: Sunday, 12th September 2021 6:21:18 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package runtime

import (
	"fmt"
	"os"
)

func GetEnv(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("required environmental variable='%s' not set", key)
	}

	return value, nil
}
