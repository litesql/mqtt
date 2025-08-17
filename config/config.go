package config

const (
	// Common config
	ClientID    = "client_id"     // Client ID
	Servers     = "servers"       // Comma-separated list of brokers servers
	Timeout     = "timeout"       // timeout in milliseconds
	KeepAlive   = "keep_alive"    // Keep alive
	Username    = "username"      // username to connect to broker
	Password    = "password"      // password to connect to broker
	CertFile    = "cert_file"     // TLS: path to certificate file
	CertKeyFile = "cert_key_file" // TLS: path to .pem certificate key file
	CertCAFile  = "ca_file"       // TLS: path to CA certificate file
	Insecure    = "insecure"      // TLS: Insecure skip TLS verification
	Logger      = "logger"        // Log errors to "stdout, stderr or file:/path/to/log.txt"

	// Subscribe module config
	TableName = "table" // table name where to store the incoming messages

	DefaultTableName          = "mqtt_data"
	DefaultPublisherVTabName  = "mqtt_pub"
	DefaultSubscriberVTabName = "mqtt_sub"
)
