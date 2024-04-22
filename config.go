package consumer

import (
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type Config struct {
	AttributeNames          []types.QueueAttributeName `json:"attributeNames"`      // ["All"]
	MaxNumberOfMessages     int32                      `json:"maxNumberOfMessages"` // 1 - 10, 1
	MessageAttributeNames   []string                   `json:"messageAttributeNames"`
	QueueUrl                *string                    `json:"queueUrl"`
	QueueName               *string                    `json:"queueName"`
	ReceiveRequestAttemptId *string                    `json:"receiveRequestAttemptId"`
	VisibilityTimeout       int32                      `json:"visibilityTimeout"` // 0 - 43200, 30
	WaitTimeSeconds         int32                      `json:"waitTimeSeconds"`   // 0 - 20, 0, 10
	ShouldDelete            bool                       `json:"shouldDelete"`

	Idle  *int64 `json:"idle"`
	Sleep *int64 `json:"sleep"`
}

func (c *Config) validate() (err error) {
	if c.QueueName == nil {
		err = errors.New("queue name is required")
	}

	return
}
