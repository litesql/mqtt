package config

import "time"

const (
	// Common config
	ClientID  = "client_id"  // Client ID
	Servers   = "servers"    // Comma-separated list of brokers servers
	Timeout   = "timeout"    // timeout in milliseconds
	KeepAlive = "keep_alive" // Keep alive
	Username  = "username"   // username to connect to broker
	Password  = "password"   // password to connect to broker

	// Subscribe module config
	TableName = "table"  // table name where to store the incoming messages
	Logger    = "logger" // Log errors to "stdout, stderr or file:/path/to/log.txt"

	DefaultTableName          = "mqtt_data"
	DefaultPublisherVTabName  = "mqtt_pub"
	DefaultSubscriberVTabName = "mqtt_sub"
	DefaultTimeout            = 10 * time.Second
)
