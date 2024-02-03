/*
 * File: consumer.go
 * Project: aws
 * File Created: Friday, 19th March 2021 9:44:16 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package awssqs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

const (
	maxMessages              = 1
	maxWaitTimeSeconds       = 10
	defaultVisibilityTimeout = 30
	maxVisibilityTimeout     = 12 * 60 * 60
	defaultWorkerPoolSize    = 30
	maxWorkerPoolSize        = 1024
)

// Consumer provides an interface for receiving messages through AWS SQS and SNS
type Consumer interface {
	// Consume polls for new messages and if it finds one, decodes it, sends it to the handler and deletes it
	//
	// A message is not considered dequeued until it has been successfully processed and deleted. There is a 30 Second
	// delay between receiving a single message and receiving the same message. This delay can be adjusted in the AWS
	// console and can also be extended during operation. If a message is successfully received 4 times but not deleted,
	// it will be considered unprocessable and sent to the DLQ automatically
	//
	// Consume uses long-polling to check and retrieve messages, if it is unable to make a connection, the aws-SDK will use its
	// advanced retrying mechanism (including exponential backoff), if all of the retries fail, then we will wait 10s before
	// trying again.
	//
	// When a new message is received, it runs in a separate go-routine that will handle the full consuming of the message, error reporting
	// and deleting
	Consume(callbackFunc func(*Message) error) error
}

// consumer is a wrapper around sqs.SQS
type consumer struct {
	sqs               *sqs.Client
	QueueURL          string
	VisibilityTimeout int32
	workerPool        int
}

// NewConsumer creates a new SQS instance and provides a configured consumer interface for
// receiving and sending messages
func NewConsumer(c *Config, queueName string) (Consumer, error) {

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRetryMode(aws.RetryModeStandard))
	if err != nil {
		return nil, err
	}

	cons := &consumer{
		sqs:               sqs.NewFromConfig(cfg),
		VisibilityTimeout: defaultVisibilityTimeout,
		workerPool:        defaultWorkerPoolSize}

	if c.VisibilityTimeout != 0 {
		if c.VisibilityTimeout > maxVisibilityTimeout {
			cons.VisibilityTimeout = maxVisibilityTimeout
		} else {
			cons.VisibilityTimeout = c.VisibilityTimeout
		}
	}

	if c.WorkerPool != 0 {
		if c.WorkerPool > maxWorkerPoolSize {
			cons.workerPool = maxWorkerPoolSize
		} else {
			cons.workerPool = c.WorkerPool
		}
		cons.workerPool = c.WorkerPool
	}

	resultURL, err := cons.sqs.GetQueueUrl(context.TODO(), &sqs.GetQueueUrlInput{
		QueueName: &queueName,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to find queue %q, %s", queueName, err.Error())
	}
	cons.QueueURL = *resultURL.QueueUrl

	return cons, nil
}

var (
	all = "All"
)

// Consume polls for new messages and if it finds one, decodes it, sends it to the handler and deletes it
//
// A message is not considered dequeued until it has been successfully processed and deleted. There is a 30 Second
// delay between receiving a single message and receiving the same message. This delay can be adjusted in the AWS
// console and can also be extended during operation. If a message is successfully received 4 times but not deleted,
// it will be considered unprocessable and sent to the DLQ automatically
//
// Consume uses long-polling to check and retrieve messages, if it is unable to make a connection, the aws-SDK will use its
// advanced retrying mechanism (including exponential backoff), if all of the retries fail, then we will wait 10s before
// trying again.
//
// When a new message is received, it runs in a separate go-routine that will handle the full consuming of the message, error reporting
// and deleting
func (c *consumer) Consume(callbackFunc func(*Message) error) error {
	jobs := make(chan *Message)
	for w := 1; w <= c.workerPool; w++ {
		go c.worker(callbackFunc, jobs)
	}

	for {
		output, err := c.sqs.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
			QueueUrl:              &c.QueueURL,
			MaxNumberOfMessages:   maxMessages,
			AttributeNames:        []types.QueueAttributeName{"SentTimestamp"},
			MessageAttributeNames: []string{all},
			VisibilityTimeout:     c.VisibilityTimeout,
			WaitTimeSeconds:       maxWaitTimeSeconds,
		})
		if err != nil {
			log.Printf("%s , retrying in 10s", err.Error())
			time.Sleep(10 * time.Second)
			continue
		}

		for _, m := range output.Messages {
			jobs <- NewMessage(&m)
		}
	}
}

// worker is an always-on concurrent worker that will take tasks when they are added into the messages buffer
func (c *consumer) worker(callbackFunc func(*Message) error, messages <-chan *Message) {
	for m := range messages {

		err := callbackFunc(m)
		if err != nil {
			log.Printf("error consuming sqs message; %s", err.Error())

			// update the visibility timeout so this message gets retried ASAP
			updatedVisibilityTimeout := int32(5)
			c.sqs.ChangeMessageVisibility(context.TODO(), &sqs.ChangeMessageVisibilityInput{
				ReceiptHandle:     m.ReceiptHandle,
				QueueUrl:          &c.QueueURL,
				VisibilityTimeout: updatedVisibilityTimeout,
			})
			continue
		}

		c.delete(m) //MESSAGE CONSUMED
	}
}

// delete will remove a message from the queue, this is necessary to fully and successfully consume a message
func (c *consumer) delete(m *Message) error {
	_, err := c.sqs.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{QueueUrl: &c.QueueURL, ReceiptHandle: m.ReceiptHandle})
	if err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}
