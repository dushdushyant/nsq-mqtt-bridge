package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTPublisher struct {
	client mqtt.Client
	config *MQTTConfig
}

func NewMQTTPublisher(config *MQTTConfig) (*MQTTPublisher, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.BrokerAddress)
	opts.SetClientID(config.ClientID)
	opts.SetUsername(config.Username)
	opts.SetPassword(config.Password)
	opts.SetCleanSession(config.CleanSession)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(10 * time.Second)
	opts.SetKeepAlive(60 * time.Second)

	// Configure TLS if enabled
	if config.UseTLS {
		tlsConfig := &tls.Config{}

		// Load CA certificate if provided
		if config.CACert != "" {
			caCert, err := ioutil.ReadFile(config.CACert)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA certificate: %w", err)
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			tlsConfig.RootCAs = caCertPool
		}

		// Load client certificate and key if provided
		if config.ClientCert != "" && config.ClientKey != "" {
			cert, err := tls.LoadX509KeyPair(config.ClientCert, config.ClientKey)
			if err != nil {
				return nil, fmt.Errorf("failed to load client certificate/key: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		tlsConfig.InsecureSkipVerify = config.SkipTLSVerify
		opts.SetTLSConfig(tlsConfig)
	}

	client := mqtt.NewClient(opts)

	return &MQTTPublisher{
		client: client,
		config: config,
	}, nil
}

func (p *MQTTPublisher) Connect() error {
	token := p.client.Connect()
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}
	log.Println("Connected to MQTT broker")
	return nil
}

func (p *MQTTPublisher) Publish(topic string, payload []byte) error {
	token := p.client.Publish(topic, p.config.QOS, p.config.Retained, payload)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish to MQTT: %w", token.Error())
	}
	return nil
}

func (p *MQTTPublisher) Disconnect() {
	p.client.Disconnect(250)
	log.Println("Disconnected from MQTT broker")
}

func (p *MQTTPublisher) IsConnected() bool {
	return p.client.IsConnected()
}
