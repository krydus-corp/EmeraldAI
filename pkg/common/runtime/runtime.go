/*
 * File: runtime.go
 * Project: runtime
 * File Created: Monday, 14th September 2020 1:09:15 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package runtime

import (
	"os"
	"os/signal"
	"syscall"
)

// WaitForExit is a function for blocking execution until a SIGINT or SIGTERM is received.
func WaitForExit(exitMessage string) {
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}
