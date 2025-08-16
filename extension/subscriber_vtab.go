package extension

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/walterwanderley/sqlite"
)

type SubscriberVirtualTable struct {
	virtualTableName string
	tableName        string
	client           mqtt.Client
	subscriptions    map[string]subscription
	stmt             *sqlite.Stmt
	stmtMu           sync.Mutex
	mu               sync.Mutex
	logger           *slog.Logger
	loggerCloser     io.Closer
}

type subscription struct {
	qos byte
}

func NewSubscriberVirtualTable(virtualTableName string, clientOptions *mqtt.ClientOptions, tableName string, conn *sqlite.Conn, loggerDef string) (*SubscriberVirtualTable, error) {

	stmt, _, err := conn.Prepare(fmt.Sprintf(`INSERT INTO %s(topic, payload, timestamp) VALUES(?, ?, ?)`, tableName))
	if err != nil {
		return nil, err
	}

	vtab := SubscriberVirtualTable{
		virtualTableName: virtualTableName,
		tableName:        tableName,
		subscriptions:    make(map[string]subscription),
		stmt:             stmt,
	}

	logger, loggerCloser, err := loggerFromConfig(loggerDef)
	if err != nil {
		return nil, err
	}
	vtab.loggerCloser = loggerCloser
	vtab.logger = logger

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

func (vt *SubscriberVirtualTable) BestIndex(in *sqlite.IndexInfoInput) (*sqlite.IndexInfoOutput, error) {
	return &sqlite.IndexInfoOutput{}, nil
}

func (vt *SubscriberVirtualTable) Open() (sqlite.VirtualCursor, error) {
	return nil, fmt.Errorf("SELECT operations on %q is not supported, exec SELECT on %q table", vt.tableName, vt.virtualTableName)
}

func (vt *SubscriberVirtualTable) Disconnect() error {
	var err error
	if vt.loggerCloser != nil {
		err = vt.loggerCloser.Close()
	}
	if size := len(vt.subscriptions); size > 0 {
		keys := make([]string, 0)
		for k := range vt.subscriptions {
			keys = append(keys, k)
		}
		tok := vt.client.Unsubscribe(keys...)
		tok.Wait()
		err = errors.Join(err, tok.Error())
	}
	vt.client.Disconnect(200)

	return errors.Join(err, vt.stmt.Finalize())
}

func (vt *SubscriberVirtualTable) Destroy() error {
	return nil
}

func (vt *SubscriberVirtualTable) Insert(values ...sqlite.Value) (int64, error) {
	if len(values) < 2 {
		return 0, fmt.Errorf("inform at least 2 values: Topic and Qos")
	}
	topic := values[0].Text()
	if topic == "" {
		return 0, fmt.Errorf("topic is invalid")
	}
	qos := values[1].Int()
	if qos < 0 || qos > 2 {
		return 0, fmt.Errorf("QoS must be the number 0, 1 or 2")
	}

	vt.mu.Lock()
	defer vt.mu.Unlock()
	if _, ok := vt.subscriptions[topic]; ok {
		return 0, fmt.Errorf("already subscribed to the %q topic", topic)
	}

	tok := vt.client.Subscribe(topic, byte(qos), vt.messageHandler)
	if tok.Wait() && tok.Error() != nil {
		return 0, fmt.Errorf("subscribe error: %w", tok.Error())
	}
	return 1, nil
}

func (vt *SubscriberVirtualTable) Update(_ sqlite.Value, _ ...sqlite.Value) error {
	return fmt.Errorf("UPDATE operations on %q is not supported", vt.virtualTableName)
}

func (vt *SubscriberVirtualTable) Replace(old sqlite.Value, new sqlite.Value, _ ...sqlite.Value) error {
	return fmt.Errorf("UPDATE operations on %q is not supported", vt.virtualTableName)
}

func (vt *SubscriberVirtualTable) Delete(v sqlite.Value) error {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	topic := v.Text()
	fmt.Println("DELETE", topic)
	if _, ok := vt.subscriptions[v.Text()]; !ok {
		return fmt.Errorf("not subscribed to the %q topic", topic)
	}
	tok := vt.client.Unsubscribe(topic)
	tok.Wait()
	err := tok.Error()
	if err == nil {
		delete(vt.subscriptions, topic)
	}
	return err
}

func (vt *SubscriberVirtualTable) messageHandler(c mqtt.Client, msg mqtt.Message) {
	vt.stmtMu.Lock()
	defer vt.stmtMu.Unlock()
	err := vt.stmt.Reset()
	if err != nil {
		vt.logger.Error("reset statement", "error", err, "topic", msg.Topic(), "message_id", msg.MessageID())
		return
	}
	vt.stmt.BindText(1, msg.Topic())
	vt.stmt.BindText(2, string(msg.Payload()))
	vt.stmt.BindText(3, time.Now().Format(time.RFC3339Nano))
	_, err = vt.stmt.Step()
	if err != nil {
		vt.logger.Error("insert data", "error", err, "topic", msg.Topic(), "message_id", msg.MessageID())
		return
	}
}

func (vt *SubscriberVirtualTable) onConnectionLost(client mqtt.Client, err error) {
	vt.logger.Error("lost connection to the broker", "virtual_table", vt.virtualTableName, "error", err)
}

func (vt *SubscriberVirtualTable) onConnectHandler(client mqtt.Client) {
	vt.logger.Debug("connected to broker", "virtual_table", vt.virtualTableName)
	for topic, subscription := range vt.subscriptions {
		client.Subscribe(topic, subscription.qos, vt.messageHandler)
	}
}
