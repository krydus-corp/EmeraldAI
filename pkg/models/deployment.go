/*
 * File: deployment.go
 * Project: models
 * File Created: Monday, 22nd August 2022 3:38:45 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package models

import "time"

type DeploymentStatus int

const (
	DeploymentStatusUnknown DeploymentStatus = iota
	DeploymentStatusInitialized
	DeploymentStatusCreating
	DeploymentStatusInService
	DeploymentStatusDeleting
	DeploymentStatusDeleted
	DeploymentStatusErr
)

func (d DeploymentStatus) String() string {
	return [...]string{"UNKNOWN", "INITIALIZED", "CREATING", "IN_SERVICE", "DELETING", "DELETED", "ERR"}[d]
}

func DeploymentStatusFromString(s string) DeploymentStatus {
	return map[string]DeploymentStatus{
		"UNKNOWN":     DeploymentStatusUnknown,
		"INITIALIZED": DeploymentStatusInitialized,
		"CREATING":    DeploymentStatusCreating,
		"IN_SERVICE":  DeploymentStatusInService,
		"DELETING":    DeploymentStatusDeleting,
		"DELETED":     DeploymentStatusDeleted,
		"ERR":         DeploymentStatusErr}[s]
}

// Deployment represents deployment domain model
//
// swagger:model Deployment
type Deployment struct {
	// Model name
	//
	ModelName string `json:"model_name" bson:"model_name"`
	// Endpoint name
	//
	EndpointName string `json:"endpoint_name" bson:"endpoint_name"`
	// Endpoint Curl
	//
	EndpointCurl string `json:"endpoint_curl" bson:"endpoint_curl"`
	// Status of this endpoint
	//
	Status string `json:"status" bson:"status"`
	// Time deployed
	//
	DeployedAt time.Time `json:"deployed_at" bson:"deployed_at"`
	// Expiration of endpoint - NOT YET IMPLEMENTED
	//
	ExpireDuration time.Duration `json:"expire_duration" bson:"expire_duration"`
	// Last error (if any) associated with the deployment
	//
	LastError *string `json:"error" bson:"error"`
}
