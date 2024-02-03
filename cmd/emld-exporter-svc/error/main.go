/*
 * File: main.go
 * Project: error
 * File Created: Monday, 22nd March 2021 7:52:01 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package main

import (
	"github.com/aws/aws-lambda-go/lambda"

	svc "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/exporter/error"
)

func main() {
	lambda.Start(svc.HandleLambdaEvent)

}
