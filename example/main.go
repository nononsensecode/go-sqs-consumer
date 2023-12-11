package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	consumer "github.com/nononsensecode/go-sqs-consumer"
	"github.com/sirupsen/logrus"
)

func Handler(message types.Message) error {
	logrus.Infof("message: %v", *message.Body)

	return nil
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	client := sqs.NewFromConfig(cfg)
	worker, err := consumer.New(client, func(c *consumer.Config) {
		c.QueueName = aws.String("event-dev-queue.fifo")
	})
	if err != nil {
		log.Fatalf("unable to create worker, %v", err)
	}

	worker.On(consumer.EventReceiveMessage, consumer.OnReceiveMessage(func(messages []types.Message) {
		fmt.Println("OnReceiveMessage", messages)
	}))
	worker.On(consumer.EventProcessMessage, consumer.OnProcessMessage(func(message types.Message) {
		fmt.Println("OnProcessMessage", message)
	}))
	worker.On(consumer.EventReceiveMessageError, consumer.OnReceiveMessageError(func(err error) {
		fmt.Println("OnReceiveMessageError", err)
	}))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go worker.Start(Handler)

	worker.Concurrent(Handler, 6)

	<-sig
}
