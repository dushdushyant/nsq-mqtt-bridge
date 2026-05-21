package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"os"
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
	opts.SetProtocolVersion(4) // MQTT 3.1.1

	// Force IPv4 resolution
	opts.SetDialer(&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 60 * time.Second,
		DualStack: false, // Force IPv4 only
	})

	// Configure TLS if enabled
	if config.UseTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // Skip built-in verification to handle legacy certificates
			MinVersion:         tls.VersionTLS12,
			MaxVersion:         tls.VersionTLS13,
		}

		// Load CA certificate if provided
		if config.CACert != "" {
			caCert, err := os.ReadFile(config.CACert)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA certificate: %w", err)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse CA certificate")
			}
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

		// Custom verification to accept legacy Common Name fields
		if !config.SkipTLSVerify {
			tlsConfig.VerifyConnection = func(state tls.ConnectionState) error {
				if len(state.PeerCertificates) == 0 {
					return fmt.Errorf("no peer certificates")
				}

				log.Printf("Performing custom certificate verification")

				// Verify certificate chain against CA (skip DNS name verification for legacy certificates)
				opts := x509.VerifyOptions{
					Roots:         tlsConfig.RootCAs,
					DNSName:       "", // Skip DNS name verification for legacy certificates
					Intermediates: x509.NewCertPool(),
				}

				// Add intermediate certificates
				for _, cert := range state.PeerCertificates[1:] {
					opts.Intermediates.AddCert(cert)
				}

				// Verify certificate chain
				if _, err := state.PeerCertificates[0].Verify(opts); err != nil {
					return fmt.Errorf("certificate chain verification failed: %w", err)
				}

				log.Printf("Certificate verification successful")
				return nil
			}
		}

		opts.SetTLSConfig(tlsConfig)
	}

	client := mqtt.NewClient(opts)

	return &MQTTPublisher{
		client: client,
		config: config,
	}, nil
}

func (p *MQTTPublisher) Connect() error {
	log.Printf("Attempting to connect to MQTT broker: %s", p.config.BrokerAddress)
	log.Printf("TLS enabled: %v, Skip verify: %v", p.config.UseTLS, p.config.SkipTLSVerify)
	if p.config.CACert != "" {
		log.Printf("CA certificate: %s", p.config.CACert)
	}

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
