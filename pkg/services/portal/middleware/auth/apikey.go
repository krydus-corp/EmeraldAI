/*
 * File: apikey.go
 * Project: auth
 * File Created: Wednesday, 6th July 2022 7:46:30 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package auth

// ApiKeyService represents API keyservice interface
type ApiKeyService interface {
	// IsWhitelistedRoute is a method for determining if an endpoint is authorized for API key authentication
	IsWhitelistedRoute(route string) bool
	// IsAuthorized determines if the `secretKey`` is an authorized key at `secretKey`
	IsAuthorized(accessKey, secretKey string) bool
}
