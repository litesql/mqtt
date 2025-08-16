# sqlite-mqtt
SQLite Extension to integrate with MQTT servers.

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

Subscriber table schema:

```sql
TABLE temp.sub(
  topic TEXT,
  qos INTEGER
)
```

### Publisher

```sh
# Create a virtual table using MQTT_PUB to configure the connection to the broker
CREATE VIRTUAL TABLE temp.pub USING mqtt_pub(servers='tcp://localhost:1883');

# Insert Topic and Payload into the created virtual table to publish messages
INSERT INTO temp.pub (topic, payload) VALUES('my/topic', 'hello');

# Insert retained messages
INSERT INTO temp.pub (topic, payload, retained) VALUES('my/topic', 'hi', 1);
```

Publisher table schema:

```sql
TABLE temp.pub(
  topic TEXT,
  payload BLOB,
  qos INTEGER, -- 0, 1 or 2
  retained INTEGER -- 0 = false, 1 = true
)
```

### Stored messages

```sh
# Set output mode (optional)
.mode qbox

# Query for the incoming messages
SELECT topic, payload, timestamp FROM mqtt_data;
┌────────────┬─────────┬───────────────────────────────────────┐
│   topic    │ payload │               timestamp               │
├────────────┼─────────┼───────────────────────────────────────┤
│ 'my/topic' │ 'hello' │ '2025-08-15T21:37:21.236886329-03:00' │
└────────────┴─────────┴───────────────────────────────────────┘

```

Incoming messages are stored in tables according to the following schema:

```sql
TABLE mqtt_data(
  client_id TEXT,
  message_id INTEGER,
  topic TEXT,
  payload BLOB,
  qos INTEGER, -- 0, 1 or 2
  retained INTEGER, -- 0 = false, 1 = true
  timestamp DATETIME
)
```

### Subscriptions management

Query the subscription virtual table (the virtual table created using **mqtt_sub**) to view all the active subscriptions for the current SQLite connection.

```sql
SELECT * FROM temp.sub;
┌─────────────┬─────┐
│    topic    │ qos │
├─────────────┼─────┤
│ 'my/topic'  │ 0   │
└─────────────┴─────┘
```

Delete the row to unsubscribe from the topic:

```sql
DELETE FROM temp.sub WHERE topic = 'my/topic';
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
