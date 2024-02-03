/*
 * File: event.go
 * Project: worker
 * File Created: Tuesday, 16th August 2022 5:09:15 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package train

import (
	"encoding/json"
	"fmt"

	awssqs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/aws/sqs"
)

type Event struct {
	ModelID string `json:"model_id"`
	UserID  string `json:"user_id"`
}

// ToJSON outputs the event in json format. Always returns.
func (e *Event) ToJSON() string {
	b, err := json.Marshal(e)
	if err != nil {
		return ""
	}
	return string(b)
}

func NewEvent(modelid, userid string) awssqs.JSONString {
	return awssqs.JSONString(fmt.Sprintf(`{"model_id":"%s","user_id":"%s"}`, modelid, userid))
}
