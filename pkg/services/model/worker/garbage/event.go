package garbage

import (
	"encoding/json"
	"fmt"

	awssqs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/aws/sqs"
)

type Action string

const (
	ActionDelete Action = "DELETE"
)

type Event struct {
	ModelID      string `json:"model_id"`
	ModelName    string `json:"model_name"`
	EndpointName string `json:"endpoint_name"`
	UserID       string `json:"user_id"`
	Action       string `json:"action"`
}

// ToJSON outputs the event in json format. Always returns.
func (e *Event) ToJSON() string {
	b, err := json.Marshal(e)
	if err != nil {
		return ""
	}
	return string(b)
}

func NewEvent(model_id, modelName, endpointName, userid string, action Action) awssqs.JSONString {
	return awssqs.JSONString(fmt.Sprintf(`{"model_id":"%s","model_name":"%s","endpoint_name":"%s","user_id":"%s","action":"%s"}`, model_id, modelName, endpointName, userid, string(action)))
}
