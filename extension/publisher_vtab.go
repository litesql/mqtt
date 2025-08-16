package extension

import (
	"fmt"
	"io"
	"log/slog"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/walterwanderley/sqlite"
)

type PublisherVirtualTable struct {
	client       mqtt.Client
	name         string
	logger       *slog.Logger
	loggerCloser io.Closer
}

func NewPublisherVirtualTable(name string, clientOptions *mqtt.ClientOptions, loggerDef string) (*PublisherVirtualTable, error) {
	vtab := PublisherVirtualTable{
		name: name,
	}

	logger, loggerCloser, err := loggerFromConfig(loggerDef)
	if err != nil {
		return nil, err
	}
	vtab.loggerCloser = loggerCloser
	vtab.logger = logger

	clientOptions.SetAutoReconnect(true)
	clientOptions.SetConnectionLostHandler(vtab.onConnectionLost)
	clientOptions.SetOnConnectHandler(vtab.onConnectHandler)

	client := mqtt.NewClient(clientOptions)

	if len(clientOptions.Servers) != 0 {
		tok := client.Connect()
		if tok.Wait() && tok.Error() != nil {
			return nil, fmt.Errorf("connecting to mqtt server: %w", tok.Error())
		}
	}

	vtab.client = client

	return &vtab, nil
}

func (vt *PublisherVirtualTable) BestIndex(in *sqlite.IndexInfoInput) (*sqlite.IndexInfoOutput, error) {
	return &sqlite.IndexInfoOutput{}, nil
}

func (vt *PublisherVirtualTable) Open() (sqlite.VirtualCursor, error) {
	return nil, fmt.Errorf("SELECT operations on %q is not supported", vt.name)
}

func (vt *PublisherVirtualTable) Disconnect() error {
	vt.client.Disconnect(200)
	return nil
}

func (vt *PublisherVirtualTable) Destroy() error {
	return nil
}

func (vt *PublisherVirtualTable) Insert(values ...sqlite.Value) (int64, error) {
	topic := values[0].Text()
	if topic == "" {
		return 0, fmt.Errorf("topic is required")
	}
	payload := values[1].Text()
	qos := values[2].Int()
	if qos < 0 || qos > 2 {
		return 0, fmt.Errorf("QoS must be the number 0, 1 or 2")
	}

	retained := values[3].Int() > 0

	tok := vt.client.Publish(topic, byte(qos), retained, payload)
	if tok.Wait() && tok.Error() != nil {
		return 0, fmt.Errorf("publisher error: %w", tok.Error())
	}

	return 1, nil
}

func (vt *PublisherVirtualTable) Update(_ sqlite.Value, _ ...sqlite.Value) error {
	return fmt.Errorf("UPDATE operations on %q is not supported", vt.name)
}

func (vt *PublisherVirtualTable) Replace(old sqlite.Value, new sqlite.Value, _ ...sqlite.Value) error {
	return fmt.Errorf("UPDATE operations on %q is not supported", vt.name)
}

func (vt *PublisherVirtualTable) Delete(_ sqlite.Value) error {
	return fmt.Errorf("DELETE operations on %q is not supported", vt.name)
}

func (vt *PublisherVirtualTable) onConnectionLost(client mqtt.Client, err error) {
	vt.logger.Error("lost connection to the broker", "virtual_table", vt.name, "error", err)
}

func (vt *PublisherVirtualTable) onConnectHandler(client mqtt.Client) {
	vt.logger.Debug("connected to broker", "virtual_table", vt.name)
}
