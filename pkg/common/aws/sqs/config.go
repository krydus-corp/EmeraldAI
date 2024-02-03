/*
 * File: config.go
 * Project: aws
 * File Created: Friday, 19th March 2021 10:00:56 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package awssqs

// Config defines the sqs configuration
type Config struct {
	// used to extend the allowed processing time of a message
	VisibilityTimeout int32
	// defines the total amount of goroutines that can be run by the consumer
	WorkerPool int
}
