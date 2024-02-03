/*
 * File: env.go
 * Project: exporter
 * File Created: Wednesday, 11th January 2023 3:44:21 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package exporter

import (
	"os"
)

const (
	// EnvQueueURL is the environment variable that contains the SQS queue URL.
	EnvQueueURL = "QUEUE_URL"
)

// QueueURL returns the SQS Queue URL set in the environment variable.
func QueueURL() string {
	return os.Getenv(EnvQueueURL)
}
