package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	configPath := flag.String("config", "config.ini", "Path to configuration file")
	flag.Parse()

	// Load configuration
	config, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Println("Configuration loaded successfully:")
	log.Println(config.String())

	// Initialize MQTT publisher
	mqttPublisher, err := NewMQTTPublisher(&config.MQTT)
	if err != nil {
		log.Fatalf("Failed to create MQTT publisher: %v", err)
	}

	// Connect to MQTT broker
	err = mqttPublisher.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer mqttPublisher.Disconnect()

	// Initialize deadletter handler
	deadletterHandler, err := NewDeadletterHandler(&config.Deadletter, config.NSQ.NSQDAddress, config.NSQ.Topic)
	if err != nil {
		log.Fatalf("Failed to create deadletter handler: %v", err)
	}
	defer deadletterHandler.Close()

	// Initialize NSQ consumer
	nsqConsumer, err := NewNSQConsumer(&config.NSQ, mqttPublisher, deadletterHandler, config.MQTT.Topic)
	if err != nil {
		log.Fatalf("Failed to create NSQ consumer: %v", err)
	}

	// Connect to NSQ
	if config.NSQ.LookupdAddresses != "" {
		err = nsqConsumer.ConnectToNSQLookupd(config.NSQ.LookupdAddresses)
		if err != nil {
			log.Fatalf("Failed to connect to NSQ lookupd: %v", err)
		}
	} else {
		err = nsqConsumer.ConnectToNSQD(config.NSQ.NSQDAddress)
		if err != nil {
			log.Fatalf("Failed to connect to NSQD: %v", err)
		}
	}

	log.Println("NSQ-MQTT Bridge is running...")
	log.Printf("Bridging NSQ topic '%s' (channel: '%s') to MQTT topic '%s'", 
		config.NSQ.Topic, config.NSQ.Channel, config.MQTT.Topic)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	nsqConsumer.Stop()
	log.Println("NSQ-MQTT Bridge stopped")
}
