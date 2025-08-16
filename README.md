# sqlite-mqtt
SQLite Extension to integrate with MQTT servers


## Installation

Download **mqtt** extension from the [releases page](https://github.com/litesql/mqtt/releases).

### Compiling from source

- [Go 1.24+](https://go.dev) is required.

```sh
go build -ldflags="-s -w" -buildmode=c-shared -o mqtt.so
```

## Basic usage

### Loading the extension

```sh
sqlite3

# Load the extension
.load ./mqtt
```

### Subscriber

```sh
# Create a virtual table using MQTT_SUB to configure the connection to the broker
CREATE VIRTUAL TABLE temp.sub USING mqtt_sub(servers='tcp://localhost:1883', table=mqtt_data);

# Insert the topic name and qos to subscribe into the created virtual table
INSERT INTO temp.sub VALUES('my/topic', 0);
```

### Publisher

```sh
# Create a virtual table using MQTT_PUB to configure the connection to the broker
CREATE VIRTUAL TABLE temp.pub USING mqtt_pub(servers='tcp://localhost:1883');

# Insert Topic, Payload and QoS into the created virtual table to publish messages
INSERT INTO temp.pub VALUES('my/topic', 'hello', 0);
```

### Stored messages

```sh

.mode qbox

SELECT * FROM mqtt_data;
┌────────────┬─────────┬───────────────────────────────────────┐
│   topic    │ payload │               timestamp               │
├────────────┼─────────┼───────────────────────────────────────┤
│ 'my/topic' │ 'hello' │ '2025-08-15T21:37:21.236886329-03:00' │
└────────────┴─────────┴───────────────────────────────────────┘

```

All incomming messages are stored in tables using the following schema:

```sql
TABLE mqtt_data(
  topic TEXT,
  payload BLOB,
  timestamp DATETIME
)
```

## Configuring

You can configure the connection to the broker by passing parameters to the VIRTUAL TABLE.

| Param | Description | Default |
|-------|-------------|---------|
| servers | Comma-separated list of URLs to connect to the broker | |
| username | Username to connect to broker | |
| password | Password to connect to broker | |
| timeout | Timeout in milliseconds | |
|	keep_alive | Keep alive| 30 |
| table | Name of the table where incoming messages will be stored. Only for mqtt_sub | mqtt_data |
| logger | Log errors to stdout, stderr or file:/path/to/file.log |
