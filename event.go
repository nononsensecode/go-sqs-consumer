package consumer

import (
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type Event string

const (
	EventReceiveMessage      Event = "ReceiveMessage"
	EventProcessMessage            = "ProcessMessage"
	EventReceiveMessageError       = "ReceiveMessageError"
)

type OnReceiveMessage func(messages []types.Message)
type OnProcessMessage func(message types.Message)
type OnReceiveMessageError func(err error)
