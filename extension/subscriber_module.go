package extension

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/walterwanderley/sqlite"

	"github.com/litesql/mqtt/config"
)

var tableNameValid = regexp.MustCompilePOSIX("^[a-zA-Z_][a-zA-Z0-9_.]*$").MatchString

type SubscriberModule struct {
}

func (m *SubscriberModule) Connect(conn *sqlite.Conn, args []string, declare func(string) error) (sqlite.VirtualTable, error) {
	virtualTableName := args[2]
	if virtualTableName == "" {
		virtualTableName = config.DefaultSubscriberVTabName
	}

	var (
		clientOptions = mqtt.NewClientOptions()

		certFilePath    string
		certKeyFilePath string
		caFilePath      string
		insecure        bool

		tableName string
		logger    string
		err       error
	)
	if len(args) > 3 {
		for _, opt := range args[3:] {
			k, v, ok := strings.Cut(opt, "=")
			if !ok {
				return nil, fmt.Errorf("invalid option: %q", opt)
			}
			k = strings.TrimSpace(k)
			v = sanitizeOptionValue(v)

			switch strings.ToLower(k) {
			case config.ClientID:
				clientOptions.ClientID = v
			case config.Timeout:
				i, err := strconv.Atoi(v)
				if err != nil {
					return nil, fmt.Errorf("invalid %q option: %w", k, err)
				}
				timeout := time.Duration(i) * time.Millisecond
				clientOptions.PingTimeout = timeout
				clientOptions.WriteTimeout = timeout
			case config.KeepAlive:
				i, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("invalid %q option: %w", k, err)
				}
				clientOptions.KeepAlive = i
			case config.Servers:
				serverList := strings.Split(v, ",")
				for _, server := range serverList {
					server = strings.TrimSpace(server)
					u, err := url.Parse(server)
					if err != nil {
						return nil, fmt.Errorf("invalid %q option: %w", k, err)
					}
					clientOptions.Servers = append(clientOptions.Servers, u)
				}
			case config.Username:
				clientOptions.Username = v
			case config.Password:
				clientOptions.Password = v
			case config.CertFile:
				certFilePath = v
			case config.CertKeyFile:
				certKeyFilePath = v
			case config.CertCAFile:
				caFilePath = v
			case config.Insecure:
				insecure, err = strconv.ParseBool(v)
				if err != nil {
					return nil, fmt.Errorf("invalid %q option: %v", k, err)
				}
			case config.TableName:
				tableName = v
			case config.Logger:
				logger = v
			}
		}
	}

	tlsConfig := tls.Config{
		InsecureSkipVerify: insecure,
	}

	if certFilePath != "" && certKeyFilePath != "" {
		clientCert, err := tls.LoadX509KeyPair(certFilePath, certKeyFilePath)
		if err != nil {
			return nil, fmt.Errorf("error loading client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}

	if caFilePath != "" {
		caCertPEM, err := os.ReadFile(caFilePath)
		if err != nil {
			return nil, fmt.Errorf("error loading CA certificate: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCertPEM) {
			return nil, fmt.Errorf("error appending CA certificate to pool")
		}
		tlsConfig.RootCAs = caCertPool
	}

	clientOptions = clientOptions.SetTLSConfig(&tlsConfig)

	if tableName == "" {
		tableName = config.DefaultTableName
	}

	if !tableNameValid(tableName) {
		return nil, fmt.Errorf("table name %q is invalid", tableName)
	}

	err = conn.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s(
	    client_id TEXT,
		message_id INTEGER,
		topic TEXT,
		payload BLOB,
		qos INTEGER,
		retained INTEGER,
		timestamp DATETIME
	)`, tableName), nil)
	if err != nil {
		return nil, fmt.Errorf("creating %q table: %w", tableName, err)
	}

	vtab, err := NewSubscriberVirtualTable(virtualTableName, clientOptions, tableName, conn, logger)
	if err != nil {
		return nil, err
	}
	return vtab, declare("CREATE TABLE x(topic TEXT PRIMARY KEY, qos INTEGER)")
}
