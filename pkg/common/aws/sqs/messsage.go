/*
 * File: messsage.go
 * Project: aws
 * File Created: Friday, 19th March 2021 10:21:17 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package awssqs

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/pkg/errors"
)

// Message serves as a wrapper for sqs.Message as well as controls the error handling channel
type Message struct {
	*types.Message
	Err chan error
}

func NewMessage(m *types.Message) *Message {
	return &Message{m, make(chan error, 1)}
}

func (m *Message) Body() []byte {
	return []byte(*m.Message.Body)
}

// Decode will unmarshal the Message into a supplied output using json
func (m *Message) Decode(out interface{}) error {
	return json.Unmarshal(m.Body(), &out)
}

// DeleteMessage deletes a message from SQS
func DeleteMessage(svc sqsiface.SQSAPI, receiptHandle, QueueURL string) (output *sqs.DeleteMessageOutput, err error) {

	input := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(QueueURL),
		ReceiptHandle: aws.String(receiptHandle),
	}

	if output, err = svc.DeleteMessage(input); err != nil {
		return nil, errors.Wrap(err, "error deleting sqs message")
	}

	return
}
