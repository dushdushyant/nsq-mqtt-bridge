package main

import (
	"fmt"
	"time"

	"gopkg.in/ini.v1"
)

type Config struct {
	NSQ        NSQConfig
	MQTT       MQTTConfig
	Deadletter DeadletterConfig
}

type NSQConfig struct {
	NSQDAddress      string
	LookupdAddresses string
	Topic            string
	Channel          string
	MaxAttempts      int
	MaxInFlight      int
	Concurrency      int
	RequeueDelay     time.Duration
}

type MQTTConfig struct {
	BrokerAddress string
	Topic         string
	ClientID      string
	Username      string
	Password      string
	QOS           byte
	Retained      bool
	CleanSession  bool
	UseTLS        bool
	CACert        string
	ClientCert    string
	ClientKey     string
	SkipTLSVerify bool
}

type DeadletterConfig struct {
	Enabled           bool
	LogFile           string
	PublishToNSQ      bool
	DeadletterTopic   string
	DeadletterChannel string
}

func LoadConfig(configPath string) (*Config, error) {
	cfg, err := ini.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	config := &Config{}

	// NSQ Config
	config.NSQ.NSQDAddress = cfg.Section("nsq").Key("nsqd_address").MustString("localhost:4150")
	config.NSQ.LookupdAddresses = cfg.Section("nsq").Key("lookupd_addresses").MustString("localhost:4161")
	config.NSQ.Topic = cfg.Section("nsq").Key("topic").MustString("")
	config.NSQ.Channel = cfg.Section("nsq").Key("channel").MustString("")
	config.NSQ.MaxAttempts = cfg.Section("nsq").Key("max_attempts").MustInt(5)
	config.NSQ.MaxInFlight = cfg.Section("nsq").Key("max_in_flight").MustInt(200)
	config.NSQ.Concurrency = cfg.Section("nsq").Key("concurrency").MustInt(10)

	requeueDelayStr := cfg.Section("nsq").Key("requeue_delay").MustString("30s")
	config.NSQ.RequeueDelay, err = time.ParseDuration(requeueDelayStr)
	if err != nil {
		return nil, fmt.Errorf("invalid requeue_delay format: %w", err)
	}

	// MQTT Config
	config.MQTT.BrokerAddress = cfg.Section("mqtt").Key("broker_address").MustString("tcp://localhost:1883")
	config.MQTT.Topic = cfg.Section("mqtt").Key("topic").MustString("")
	config.MQTT.ClientID = cfg.Section("mqtt").Key("client_id").MustString("nsq-mqtt-bridge")
	config.MQTT.Username = cfg.Section("mqtt").Key("username").MustString("")
	config.MQTT.Password = cfg.Section("mqtt").Key("password").MustString("")
	config.MQTT.QOS = byte(cfg.Section("mqtt").Key("qos").MustInt(1))
	config.MQTT.Retained = cfg.Section("mqtt").Key("retained").MustBool(false)
	config.MQTT.CleanSession = cfg.Section("mqtt").Key("clean_session").MustBool(true)
	config.MQTT.UseTLS = cfg.Section("mqtt").Key("use_tls").MustBool(false)
	config.MQTT.CACert = cfg.Section("mqtt").Key("ca_cert").MustString("")
	config.MQTT.ClientCert = cfg.Section("mqtt").Key("client_cert").MustString("")
	config.MQTT.ClientKey = cfg.Section("mqtt").Key("client_key").MustString("")
	config.MQTT.SkipTLSVerify = cfg.Section("mqtt").Key("skip_tls_verify").MustBool(false)

	// Deadletter Config
	config.Deadletter.Enabled = cfg.Section("deadletter").Key("enabled").MustBool(true)
	config.Deadletter.LogFile = cfg.Section("deadletter").Key("log_file").MustString("deadletter.log")
	config.Deadletter.PublishToNSQ = cfg.Section("deadletter").Key("publish_to_nsq").MustBool(false)
	config.Deadletter.DeadletterTopic = cfg.Section("deadletter").Key("deadletter_topic").MustString("deadletter_topic")
	config.Deadletter.DeadletterChannel = cfg.Section("deadletter").Key("deadletter_channel").MustString("deadletter_channel")

	// Validate required fields
	if config.NSQ.Topic == "" {
		return nil, fmt.Errorf("nsq topic is required")
	}
	if config.NSQ.Channel == "" {
		return nil, fmt.Errorf("nsq channel is required")
	}
	if config.MQTT.Topic == "" {
		return nil, fmt.Errorf("mqtt topic is required")
	}
	if config.MQTT.BrokerAddress == "" {
		return nil, fmt.Errorf("mqtt broker_address is required")
	}

	return config, nil
}

func (c *Config) String() string {
	return fmt.Sprintf(
		"NSQ: Topic=%s, Channel=%s, MaxAttempts=%d, MaxInFlight=%d, Concurrency=%d\n"+
			"MQTT: Broker=%s, Topic=%s, ClientID=%s, QOS=%d, UseTLS=%v\n"+
			"Deadletter: Enabled=%v, LogFile=%s, PublishToNSQ=%v",
		c.NSQ.Topic, c.NSQ.Channel, c.NSQ.MaxAttempts, c.NSQ.MaxInFlight, c.NSQ.Concurrency,
		c.MQTT.BrokerAddress, c.MQTT.Topic, c.MQTT.ClientID, c.MQTT.QOS, c.MQTT.UseTLS,
		c.Deadletter.Enabled, c.Deadletter.LogFile, c.Deadletter.PublishToNSQ,
	)
}
