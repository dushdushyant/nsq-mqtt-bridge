# NSQ-MQTT Bridge

A Go application that bridges messages from NSQ to MQTT, with support for deadletter handling and certificate-based MQTT authentication.

## Features

- **NSQ Consumer**: Consumes messages from NSQ topics with configurable options:
  - Max attempts
  - Max in-flight messages
  - Concurrency
  - Requeue delay
- **MQTT Publisher**: Publishes messages to MQTT topics with:
  - Certificate-based TLS authentication
  - Configurable QoS, retained messages, and clean session
  - Auto-reconnect support
- **Deadletter Handling**: Failed messages are handled with:
  - File logging (JSON format)
  - Optional publishing to NSQ deadletter channel
- **Configuration**: INI-based configuration file

## Prerequisites

- Go 1.16 or higher
- NSQ server (nsqd and optionally nsqlookupd)
- MQTT broker (e.g., Mosquitto, EMQX, etc.)

## Installation

1. Clone or navigate to the project directory:
```bash
cd c:/Users/dushd/OneDrive/Desktop/code/nsq
```

2. Install dependencies:
```bash
go mod download
```

3. Build the application:
```bash
go build -o nsq-mqtt-bridge
```

## Configuration

Edit `config.ini` to configure the bridge:

### NSQ Section
```ini
[nsq]
nsqd_address = localhost:4150
lookupd_addresses = localhost:4161
topic = input_topic
channel = input_channel
max_attempts = 5
max_in_flight = 200
concurrency = 10
requeue_delay = 30s
```

### MQTT Section
```ini
[mqtt]
broker_address = tcp://localhost:1883
topic = output/topic
client_id = nsq-mqtt-bridge
username = 
password = 
qos = 1
retained = false
clean_session = true
# Certificate-based authentication
use_tls = false
ca_cert = 
client_cert = 
client_key = 
skip_tls_verify = false
```

### Deadletter Section
```ini
[deadletter]
enabled = true
log_file = deadletter.log
publish_to_nsq = false
deadletter_topic = deadletter_topic
deadletter_channel = deadletter_channel
```

## Certificate-Based MQTT Authentication

To use certificate-based authentication with MQTT:

1. Set `use_tls = true` in the MQTT section
2. Provide paths to your certificates:
   - `ca_cert`: Path to CA certificate (PEM format)
   - `client_cert`: Path to client certificate (PEM format)
   - `client_key`: Path to client private key (PEM format)
3. Set `skip_tls_verify = false` for production use

Example:
```ini
[mqtt]
use_tls = true
ca_cert = /path/to/ca.crt
client_cert = /path/to/client.crt
client_key = /path/to/client.key
skip_tls_verify = false
```

## Running the Application

Run with default config file:
```bash
go run main.go
```

Or specify a custom config file:
```bash
go run main.go -config /path/to/config.ini
```

Or run the compiled binary:
```bash
./nsq-mqtt-bridge
```

## Deadletter Handling

When MQTT publishing fails, messages are handled as follows:

1. **File Logging**: Failed messages are logged to `deadletter.log` in JSON format with:
   - Timestamp
   - Original message
   - Error details
   - Attempt count
   - MQTT topic
   - Metadata

2. **NSQ Deadletter Channel** (optional): If `publish_to_nsq = true`, failed messages are also published to the configured NSQ deadletter topic/channel.

## Message Flow

1. NSQ consumer receives message from NSQ topic/channel
2. Message is published to MQTT topic
3. If MQTT publish fails:
   - Message is logged to deadletter file
   - Optionally published to NSQ deadletter channel
   - Message is requeued (if max attempts not reached)
4. If max attempts reached, message is not requeued

## Stopping the Application

Press `Ctrl+C` to gracefully stop the bridge. The application will:
- Stop the NSQ consumer
- Disconnect from MQTT broker
- Close deadletter log file
- Stop NSQ deadletter producer (if enabled)

## Dependencies

- `github.com/eclipse/paho.mqtt.golang` - MQTT client library
- `github.com/nsqio/go-nsq` - NSQ client library
- `gopkg.in/ini.v1` - INI file parsing

## License

This project is provided as-is for use in bridging NSQ and MQTT systems.
