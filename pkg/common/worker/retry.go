/*
 * File: retry.go
 * Project: worker
 * File Created: Sunday, 11th September 2022 4:03:01 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package worker

import (
	"math/rand"
	"time"
)

type Stop struct {
	error
}

func Retry[T any](attempts int, backoff, retryJitter bool, sleep time.Duration, f func() (T, error)) (T, error) {
	v, err := f()

	if err != nil {
		if s, ok := err.(Stop); ok {
			return v, s.error
		}

		if attempts--; attempts > 0 {
			if retryJitter {
				jitter := time.Duration(rand.Int63n(int64(sleep)))
				time.Sleep(sleep + jitter/2)
			} else {
				time.Sleep(sleep)
			}

			if backoff {
				return Retry(attempts, backoff, retryJitter, 2*sleep, f)
			}
			return Retry(attempts, backoff, retryJitter, sleep, f)
		}
		return v, err
	}
	return v, nil
}
