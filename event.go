package consumer

import (
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

const (
	EventReceiveMessage      string = "ReceiveMessage"
	EventProcessMessage      string = "ProcessMessage"
	EventReceiveMessageError string = "ReceiveMessageError"
)

type OnReceiveMessage func(messages []types.Message)
type OnProcessMessage func(message types.Message)
type OnReceiveMessageError func(err error)
