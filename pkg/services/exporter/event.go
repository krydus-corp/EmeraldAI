/*
 * File: event.go
 * Project: exporter
 * File Created: Thursday, 5th January 2023 8:27:21 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package exporter

// Event models an exporter event
type Event struct {
	ExportID string `json:"export_id"`
	UserID   string `json:"user_id"`

	Error string `json:"error"`
}
