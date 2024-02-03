/*
 * File: enforcer.go
 * Project: rbac
 * File Created: Sunday, 14th March 2021 3:14:41 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package auth

import (
	"encoding/csv"
	"os"
	"strings"

	casbin "github.com/casbin/casbin/v2"
	mongo_adapter "github.com/casbin/mongodb-adapter/v3"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
)

type Enforcer struct {
	*casbin.Enforcer
}

func NewEnforcer(modelFile, policyFile, mongoURL string) (*Enforcer, error) {
	// Initialize a MongoDB adapter and use it in a Casbin enforcer:
	// The adapter will use the database named "casbin".
	// If it doesn't exist, the adapter will create it automatically.
	adapter, err := mongo_adapter.NewAdapter(mongoURL)
	if err != nil {
		return nil, err
	}

	enforcer, err := casbin.NewEnforcer(modelFile, adapter)
	if err != nil {
		return nil, err
	}

	// Load the policies from DB.
	enforcer.LoadPolicy()
	if err != nil {
		return nil, err
	}

	// Load default policies
	namedPolicies, err := parsePolicyFile(policyFile)
	if err != nil {
		return nil, err
	}

	existingPolicies := enforcer.GetNamedPolicy("p")
	for _, policy := range namedPolicies {
		if !hasPolicy(existingPolicies, policy) {
			_, err = enforcer.AddNamedPolicy("p", policy)
			if err != nil {
				return nil, err
			}
		}
	}

	// Save the policy back to DB.
	enforcer.SavePolicy()
	if err != nil {
		return nil, err
	}

	// Load the policies from DB.
	enforcer.LoadPolicy()
	if err != nil {
		return nil, err
	}

	return &Enforcer{
		enforcer,
	}, nil
}

func parsePolicyFile(policyFile string) ([][]string, error) {
	f, err := os.Open(policyFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)

	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	namedPolicies := [][]string{}
	for _, record := range records {
		if record[0] == "p" {
			namedPolicies = append(namedPolicies, record[1:])
		}
	}

	return namedPolicies, nil
}

func hasPolicy(existingPolicies [][]string, policy []string) bool {
loop:
	for _, p := range existingPolicies {
		if len(p) != len(policy) {
			continue
		}

		log.Debugf("checking policy %v against %v", policy, p)
		for idx, elem := range p {
			if strings.TrimSpace(policy[idx]) != strings.TrimSpace(elem) {
				continue loop
			}
		}
		return true
	}
	return false
}
