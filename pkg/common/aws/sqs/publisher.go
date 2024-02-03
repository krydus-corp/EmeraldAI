/*
 * File: publisher.go
 * Project: sqs
 * File Created: Friday, 12th November 2021 12:59:20 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package awssqs

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
)

// Publisher is the interface clients can use to publish messages
type Publisher interface {
	Publish(ctx context.Context, msg json.Marshaler, messageGroupId ...string) error
}

// Publisher is the AWS SNS message publisher
type publisher struct {
	sqs      *sqs.Client
	QueueURL string
}

// Publish allows SQS Publisher to implement the publisher.Publisher interface
// and publish messages to an AWS SQS backend
func (p *publisher) Publish(ctx context.Context, msg json.Marshaler, messageGroupId ...string) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	input := &sqs.SendMessageInput{
		MessageBody: aws.String(string(b)),
		QueueUrl:    &p.QueueURL,
	}

	// Required parameters for fifo queues
	if strings.HasSuffix(p.QueueURL, ".fifo") {
		hash := sha256.New()
		if _, err := hash.Write(b); err != nil {
			return err
		}
		input.MessageDeduplicationId = common.Ptr(fmt.Sprintf("%x", hash.Sum(nil)))

		input.MessageGroupId = common.Ptr("default")
		if len(messageGroupId) > 0 {
			if messageGroupId[0] != "" {
				input.MessageGroupId = common.Ptr(messageGroupId[0])
			}
		}
	}

	_, err = p.sqs.SendMessage(ctx, input)

	return err
}

// New creates a new AWS SQS publisher
func NewPublisher(c *Config, queueName string) (Publisher, error) {

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRetryMode(aws.RetryModeStandard))
	if err != nil {
		return nil, err
	}

	pub := &publisher{
		sqs: sqs.NewFromConfig(cfg),
	}

	resultURL, err := pub.sqs.GetQueueUrl(context.TODO(), &sqs.GetQueueUrlInput{
		QueueName: &queueName,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to queue %q, %s", queueName, err.Error())
	}
	pub.QueueURL = *resultURL.QueueUrl

	return pub, nil
}
