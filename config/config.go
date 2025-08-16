package config

const (
	// Common config
	ClientID  = "client_id"  // Client ID
	Servers   = "servers"    // Comma-separated list of brokers servers
	Timeout   = "timeout"    // timeout in milliseconds
	KeepAlive = "keep_alive" // Keep alive
	Username  = "username"   // username to connect to broker
	Password  = "password"   // password to connect to broker
	Logger    = "logger"     // Log errors to "stdout, stderr or file:/path/to/log.txt"

	// Subscribe module config
	TableName = "table" // table name where to store the incoming messages

	DefaultTableName          = "mqtt_data"
	DefaultPublisherVTabName  = "mqtt_pub"
	DefaultSubscriberVTabName = "mqtt_sub"
)
