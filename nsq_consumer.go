package main

import (
	"log"
	"time"

	"github.com/nsqio/go-nsq"
)

type NSQConsumer struct {
	consumer *nsq.Consumer
	config   *NSQConfig
	handler  *MessageHandler
}

type MessageHandler struct {
	mqttPublisher *MQTTPublisher
	deadletter    *DeadletterHandler
	mqttTopic     string
	config        *NSQConfig
}

func NewNSQConsumer(config *NSQConfig, mqttPublisher *MQTTPublisher, deadletter *DeadletterHandler, mqttTopic string) (*NSQConsumer, error) {
	nsqConfig := nsq.NewConfig()
	nsqConfig.MaxAttempts = uint16(config.MaxAttempts)
	nsqConfig.MaxInFlight = config.MaxInFlight
	nsqConfig.DefaultRequeueDelay = config.RequeueDelay
	nsqConfig.MsgTimeout = 2 * time.Minute
	nsqConfig.MaxBackoffDuration = 60 * time.Second

	handler := &MessageHandler{
		mqttPublisher: mqttPublisher,
		deadletter:    deadletter,
		mqttTopic:     mqttTopic,
		config:        config,
	}

	consumer, err := nsq.NewConsumer(config.Topic, config.Channel, nsqConfig)
	if err != nil {
		return nil, err
	}

	consumer.AddHandler(nsq.HandlerFunc(handler.HandleMessage))

	return &NSQConsumer{
		consumer: consumer,
		config:   config,
		handler:  handler,
	}, nil
}

func (c *NSQConsumer) ConnectToNSQLookupd(lookupdAddresses string) error {
	err := c.consumer.ConnectToNSQLookupd(lookupdAddresses)
	if err != nil {
		return err
	}
	log.Printf("Connected to NSQ lookupd: %s", lookupdAddresses)
	return nil
}

func (c *NSQConsumer) ConnectToNSQD(nsqdAddress string) error {
	err := c.consumer.ConnectToNSQD(nsqdAddress)
	if err != nil {
		return err
	}
	log.Printf("Connected to NSQD: %s", nsqdAddress)
	return nil
}

func (c *NSQConsumer) Stop() {
	c.consumer.Stop()
	log.Println("NSQ consumer stopped")
}

func (h *MessageHandler) HandleMessage(message *nsq.Message) error {
	messageBody := string(message.Body)
	log.Printf("Received NSQ message (attempt %d): %s", message.Attempts, messageBody)

	// Try to publish to MQTT
	err := h.mqttPublisher.Publish(h.mqttTopic, message.Body)
	if err != nil {
		log.Printf("Failed to publish to MQTT: %v", err)

		// Handle deadletter
		if h.deadletter != nil {
			dlErr := h.deadletter.Handle(messageBody, h.mqttTopic, err, int(message.Attempts))
			if dlErr != nil {
				log.Printf("Failed to handle deadletter: %v", dlErr)
			}
		}

		// Requeue the message if max attempts not reached
		if int(message.Attempts) < h.config.MaxAttempts {
			log.Printf("Requeueing message (attempt %d/%d)", message.Attempts, h.config.MaxAttempts)
			return err
		}

		// Max attempts reached, don't requeue
		log.Printf("Max attempts (%d) reached, message will not be requeued", h.config.MaxAttempts)
		return nil
	}

	log.Printf("Successfully published to MQTT topic: %s", h.mqttTopic)
	return nil
}
