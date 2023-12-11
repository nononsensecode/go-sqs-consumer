package consumer

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type Handler func(message types.Message) error

type Worker struct {
	Config *Config
	Input  *sqs.ReceiveMessageInput
	Sqs    SQSAPI
	Events map[Event]interface{}
}

type Opt func(*Config)

func New(sqsClient SQSAPI, opts ...Opt) (*Worker, error) {
	if sqsClient == nil {
		err := errors.New("sqs client is required")
		return nil, err
	}

	config := &Config{
		AttributeNames:      []types.QueueAttributeName{"All"},
		MaxNumberOfMessages: 1,
		VisibilityTimeout:   30,
		WaitTimeSeconds:     5,
		Idle:                aws.Int64(0),
		Sleep:               aws.Int64(0),
	}

	for _, opt := range opts {
		opt(config)
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	worker := &Worker{
		Config: config,
		Sqs:    sqsClient,
		Events: make(map[Event]interface{}),
	}

	worker.SetConfig(config)
	worker.setQueueUrl(*config.QueueName)
	return worker, nil
}

func (w *Worker) SetConfig(config *Config) {
	if config == nil {
		w.Config = &Config{}
		config = w.Config
	}

	if config.Idle == nil {
		config.Idle = aws.Int64(0)
	}
	if config.Sleep == nil {
		config.Sleep = aws.Int64(0)
	}

	if config.AttributeNames == nil {
		config.AttributeNames = []types.QueueAttributeName{"All"}
	}
	if config.MaxNumberOfMessages == 0 {
		config.MaxNumberOfMessages = 1
	}
	if config.VisibilityTimeout == 0 {
		config.VisibilityTimeout = 30
	}
	if config.WaitTimeSeconds == 0 {
		config.WaitTimeSeconds = 5
	}

	w.Input = &sqs.ReceiveMessageInput{
		AttributeNames:          config.AttributeNames,
		MaxNumberOfMessages:     config.MaxNumberOfMessages,
		MessageAttributeNames:   config.MessageAttributeNames,
		ReceiveRequestAttemptId: config.ReceiveRequestAttemptId,
		VisibilityTimeout:       config.VisibilityTimeout,
		WaitTimeSeconds:         config.WaitTimeSeconds,
	}
}

func (w *Worker) SetAttributeNames(attributeNames []string) {
	w.Input.AttributeNames = make([]types.QueueAttributeName, len(attributeNames))
	for i := 0; i < len(attributeNames); i++ {
		w.Input.AttributeNames[i] = types.QueueAttributeName(attributeNames[i])
	}
}

func (w *Worker) SetMaxNumberOfMessages(maxNumberOfMessages int32) {
	w.Input.MaxNumberOfMessages = maxNumberOfMessages
}

func (w *Worker) SetMessageAttributeNames(messageAttributeNames []string) {
	if len(messageAttributeNames) == 0 {
		w.Input.MessageAttributeNames = nil
	} else {
		w.Input.MessageAttributeNames = make([]string, len(messageAttributeNames))
		copy(w.Input.MessageAttributeNames, messageAttributeNames)
	}
}

func (w *Worker) setQueueUrl(queueUrl string) (err error) {
	urlInput := sqs.GetQueueUrlInput{
		QueueName: w.Config.QueueName,
	}
	urlOutput, err := w.Sqs.GetQueueUrl(context.Background(), &urlInput)
	if err != nil {
		return
	}

	w.Input.QueueUrl = urlOutput.QueueUrl
	return
}

func (w *Worker) SetReceiveRequestAttemptId(receiveRequestAttemptId string) {
	if len(receiveRequestAttemptId) == 0 {
		w.Input.ReceiveRequestAttemptId = nil
	} else {
		w.Input.ReceiveRequestAttemptId = aws.String(receiveRequestAttemptId)
	}
}

func (w *Worker) SetVisibilityTimeout(visibilityTimeout int32) {
	w.Input.VisibilityTimeout = visibilityTimeout
}

func (w *Worker) SetWaitTimeSeconds(waitTimeSeconds int32) {
	w.Input.WaitTimeSeconds = waitTimeSeconds
}

func (w *Worker) SetSqs(sqs SQSAPI) {
	w.Sqs = sqs
}

func (w *Worker) On(event Event, callback any) {
	switch event {
	case EventReceiveMessage:
		cb, _ := callback.(OnReceiveMessage)
		if cb == nil {
			panic(errors.New("error OnReceiveMessage"))
		}
	case EventProcessMessage:
		cb, _ := callback.(OnProcessMessage)
		if cb == nil {
			panic(errors.New("error OnProcessMessage"))
		}
	case EventReceiveMessageError:
		cb, _ := callback.(OnReceiveMessageError)
		if cb == nil {
			panic(errors.New("error OnReceiveMessageError"))
		}
	default:
		panic(errors.New("error event"))
	}

	w.Events[event] = callback
}

func (w *Worker) Start(handler Handler) {
	for idle := int64(0); ; {
		if *w.Config.Idle > 0 && idle > *w.Config.Idle && *w.Config.Sleep > 0 {
			idle = 0

			time.Sleep(time.Duration(*w.Config.Sleep) * time.Second)
		}

		output, err := w.Sqs.ReceiveMessage(context.TODO(), w.Input)
		if err != nil {
			if err != nil {
				log.Printf("[SQS] ReceiveMessage error: %v %v", err, output)
			}

			if w.Events[EventReceiveMessageError] != nil {
				w.Events[EventReceiveMessageError].(OnReceiveMessageError)(err)
			}

			continue
		}

		if len(output.Messages) > 0 {
			idle = 0

			if w.Events[EventReceiveMessage] != nil {
				w.Events[EventReceiveMessage].(OnReceiveMessage)(output.Messages)
			}

			w.run(handler, output.Messages)
		} else {
			idle++
		}
	}
}

func (w *Worker) Concurrent(handler Handler, concurrency int) {
	for i := 0; i < concurrency; i++ {
		go w.Start(handler)
	}
}

func (w *Worker) run(handler Handler, messages []types.Message) {
	var wg sync.WaitGroup
	wg.Add(len(messages))

	for i := range messages {
		go func(message types.Message) {
			defer wg.Done()

			if err := w.handleMessage(message, handler); err != nil {
				log.Printf("[SQS] handleMessage error: %v", err)
				return
			}

			if w.Events[EventProcessMessage] != nil {
				w.Events[EventProcessMessage].(OnProcessMessage)(message)
			}
		}(messages[i])
	}

	wg.Wait()
}

func (w *Worker) handleMessage(message types.Message, handler Handler) error {
	if err := handler(message); err != nil {
		return err
	}

	input := &sqs.DeleteMessageInput{
		QueueUrl:      w.Input.QueueUrl,
		ReceiptHandle: message.ReceiptHandle,
	}

	output, err := w.Sqs.DeleteMessage(context.TODO(), input)
	if err != nil {
		log.Printf("[SQS] DeleteMessage error: %v %v", err, output)
	}

	return nil
}
