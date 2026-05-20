package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/nsqio/go-nsq"
)

type DeadletterHandler struct {
	config      *DeadletterConfig
	file        *os.File
	mu          sync.Mutex
	nsqProducer *nsq.Producer
	nsqdAddress string
	nsqTopic    string
}

type DeadletterMessage struct {
	Timestamp   time.Time              `json:"timestamp"`
	NSQMessage  string                 `json:"nsq_message"`
	Error       string                 `json:"error"`
	Attempts    int                    `json:"attempts"`
	MQTTTopic   string                 `json:"mqtt_topic"`
	MessageBody string                 `json:"message_body"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func NewDeadletterHandler(config *DeadletterConfig, nsqdAddress string, nsqTopic string) (*DeadletterHandler, error) {
	dh := &DeadletterHandler{
		config:      config,
		nsqdAddress: nsqdAddress,
		nsqTopic:    nsqTopic,
	}

	if config.Enabled {
		// Open log file for appending
		file, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open deadletter log file: %w", err)
		}
		dh.file = file
		log.Printf("Deadletter handler initialized with log file: %s", config.LogFile)

		// Initialize NSQ producer if publish_to_nsq is enabled
		if config.PublishToNSQ {
			nsqConfig := nsq.NewConfig()
			producer, err := nsq.NewProducer(nsqdAddress, nsqConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to create NSQ producer for deadletter: %w", err)
			}
			dh.nsqProducer = producer
			log.Printf("Deadletter NSQ producer initialized for topic: %s", config.DeadletterTopic)
		}
	}

	return dh, nil
}

func (dh *DeadletterHandler) Handle(messageBody string, mqttTopic string, err error, attempts int) error {
	if !dh.config.Enabled {
		return nil
	}

	dlMessage := DeadletterMessage{
		Timestamp:   time.Now(),
		NSQMessage:  messageBody,
		Error:       err.Error(),
		Attempts:    attempts,
		MQTTTopic:   mqttTopic,
		MessageBody: messageBody,
		Metadata: map[string]interface{}{
			"nsq_topic": dh.nsqTopic,
		},
	}

	// Log to file
	if dh.file != nil {
		dh.mu.Lock()
		defer dh.mu.Unlock()

		jsonData, err := json.Marshal(dlMessage)
		if err != nil {
			log.Printf("Failed to marshal deadletter message: %v", err)
			return err
		}

		if _, err := dh.file.WriteString(string(jsonData) + "\n"); err != nil {
			log.Printf("Failed to write to deadletter log file: %v", err)
			return err
		}
		log.Printf("Deadletter message logged to file: %s", dh.config.LogFile)
	}

	// Publish to NSQ deadletter channel if enabled
	if dh.config.PublishToNSQ && dh.nsqProducer != nil {
		jsonData, err := json.Marshal(dlMessage)
		if err != nil {
			log.Printf("Failed to marshal deadletter message for NSQ: %v", err)
			return err
		}

		err = dh.nsqProducer.Publish(dh.config.DeadletterTopic, jsonData)
		if err != nil {
			log.Printf("Failed to publish to NSQ deadletter topic: %v", err)
			return err
		}
		log.Printf("Deadletter message published to NSQ topic: %s", dh.config.DeadletterTopic)
	}

	return nil
}

func (dh *DeadletterHandler) Close() {
	if dh.file != nil {
		dh.file.Close()
		log.Println("Deadletter log file closed")
	}

	if dh.nsqProducer != nil {
		dh.nsqProducer.Stop()
		log.Println("Deadletter NSQ producer stopped")
	}
}
