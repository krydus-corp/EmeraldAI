/*
 * File: query.go
 * Project: models
 * File Created: Monday, 20th September 2021 5:49:02 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package models

import (
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"gopkg.in/mgo.v2/bson"
)

// QueryReq models parameters used for querying other models.
// Example:
//
//	{
//		"limit": 10,
//		"page": 0,
//		"sort_key": "created_at",
//		"sort_val": 1,
//		"operator": "and",
//		"filters": [
//		  {
//			"key": "name",
//			"regex": {
//			  "enable": true,
//			  "options": "i"
//			},
//			"value": "japan"
//		  },
//		  {
//			"key": "name",
//			"regex": {
//			  "enable": true,
//			  "options": "i"
//			},
//			"value": "drone"
//		  }
//		]
//	  }
type QueryReq struct {
	Filters  []Filter `json:"filters" validate:"omitempty"`
	Operator string   `json:"operator" validate:"omitempty,oneof=and or"`

	PaginationReq
}

// Transform checks and converts http query into database query model
func (q QueryReq) Transform() Query {

	// Logical operator defaults to `and`
	operater := "and"
	if q.Operator != "" {
		operater = q.Operator
	}

	return Query{
		Filters:    q.Filters,
		Operator:   operater,
		Pagination: q.PaginationReq.Transform(),
	}
}

// Query is the final Query parameters model after being transformed from the request
type Query struct {
	Filters  []Filter
	Operator string

	Pagination
}

// NewQueryFilter constructs the Mongo BSON document filter
func (q *Query) NewQueryFilter(userid string) (interface{}, error) {
	filters := []interface{}{}
	for _, f := range q.Filters {

		// Validate filter
		if err := f.Validate(); err != nil {
			return nil, err
		}
		switch {
		// Regex filter
		case f.Regex.Enable:
			filters = append(filters, bson.M{f.Key: bson.M{"$regex": primitive.Regex{Pattern: f.Value.(string), Options: f.Regex.Options}}})
		// Datetime
		case f.Datetime.Enable:
			filters = append(filters, bson.M{f.Key: bson.M{f.Datetime.Options: primitive.NewDateTimeFromTime(f.Datetime.datetime)}})
		// All others
		default:
			filters = append(filters, map[string]interface{}{f.Key: f.Value})
		}
	}

	return bson.M{
		"$and": []interface{}{
			bson.M{"userid": userid},
			bson.M{"$" + q.Operator: filters},
		},
	}, nil
}

// Filter models query filter parameters
type Filter struct {
	Key   string      `json:"key" validate:"required"`
	Value interface{} `json:"value" validate:"required"`
	Regex struct {
		Enable  bool   `json:"enable" validate:"omitempty"`
		Options string `json:"options" validate:"omitempty"`
	} `json:"regex" validate:"omitempty"`
	Datetime struct {
		Enable   bool      `json:"enable" validate:"omitempty"`
		Options  string    `json:"options" validate:"omitempty"`
		datetime time.Time // parsed during validation
	} `json:"datetime" validate:"omitempty"`
}

// Validate validates the filters and options in a Query
func (f *Filter) Validate() error {
	if f.Regex.Enable {
		_, ok := f.Value.(string)
		if !ok {
			return fmt.Errorf("regex filter must be string type")
		}

		if !validRegexOptions(f.Regex.Options) {
			return fmt.Errorf("invalid regex options; acceptable options include: `i`, `m`, `x`, `s`")
		}
	}

	if f.Datetime.Enable {
		// Datetime enabled
		v, ok := f.Value.(string)
		if !ok {
			return fmt.Errorf("datetime filter must be string type")
		}
		// Validate key
		k := strings.ToLower(f.Key)
		if k != "created_at" && k != "updated_at" {
			return fmt.Errorf("invalid datetime filter; acceptable keys include: `created_at`, `updated_at`")
		}
		// Validate options
		o := strings.ToLower(f.Datetime.Options)
		if o != "$gt" && o != "$gte" && o != "$lt" && o != "$lte" {
			return fmt.Errorf("invalid datetime options; acceptable options include: `$gt`, `$gte`, `$lt`, `$lte`")
		}
		// Atempt to parse as datetime
		initDate, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return fmt.Errorf("unsupported format for datetime value=%s; err=%s", v, err.Error())
		}

		f.Datetime.datetime = initDate
	}

	return nil
}

func validRegexOptions(options string) bool {
	options = strings.ToLower(options)
	for _, r := range options {
		if r != 'i' && r != 'm' && r != 'x' && r != 's' {
			return false
		}
	}
	return true
}
