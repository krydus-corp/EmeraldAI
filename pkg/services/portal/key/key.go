/*
 * File: key.go
 * Project: key
 * File Created: Wednesday, 6th July 2022 8:06:59 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package key

import (
	"strings"

	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

// New generates new API key service necessary for auth middleware
func New(db *db.DB, platform *platform.Platform, whitelist []string) (*Service, error) {

	return &Service{
		whitelist: whitelist,
		db:        db,
		platform:  platform,
	}, nil
}

// Service provides an API key implementation
type Service struct {
	// Whitelisted API key routes
	whitelist []string

	db       *db.DB
	platform *platform.Platform
}

func (s *Service) IsWhitelistedRoute(route string) bool {
	for _, r := range s.whitelist {
		if strings.Contains(route, r) {
			return true
		}
	}
	return false
}

func (s *Service) IsAuthorized(accessKey, secretKey string) bool {

	// accessKey in this context is the userid and secretKey is the API key associated with the user
	user, err := s.platform.UserDB.View(s.db, accessKey)
	if err != nil {
		return false
	}

	if user.APIKey == secretKey {
		return true
	}

	return false
}
