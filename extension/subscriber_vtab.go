package extension

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/walterwanderley/sqlite"
)

type SubscriberVirtualTable struct {
	virtualTableName string
	tableName        string
	client           mqtt.Client
	subscriptions    []subscription
	stmt             *sqlite.Stmt
	stmtMu           sync.Mutex
	mu               sync.Mutex
	logger           *slog.Logger
	loggerCloser     io.Closer
}

type subscription struct {
	topic string
	qos   byte
}

func NewSubscriberVirtualTable(virtualTableName string, clientOptions *mqtt.ClientOptions, tableName string, conn *sqlite.Conn, loggerDef string) (*SubscriberVirtualTable, error) {

	stmt, _, err := conn.Prepare(fmt.Sprintf(`INSERT INTO %s(topic, payload, timestamp) VALUES(?, ?, ?)`, tableName))
	if err != nil {
		return nil, err
	}

	vtab := SubscriberVirtualTable{
		virtualTableName: virtualTableName,
		tableName:        tableName,
		subscriptions:    make([]subscription, 0),
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
	return &sqlite.IndexInfoOutput{EstimatedCost: 1000000}, nil
}

func (vt *SubscriberVirtualTable) Open() (sqlite.VirtualCursor, error) {
	return newSubscriptionsCursor(vt.subscriptions), nil
}

func (vt *SubscriberVirtualTable) Disconnect() error {
	var err error
	if vt.loggerCloser != nil {
		err = vt.loggerCloser.Close()
	}
	if size := len(vt.subscriptions); size > 0 {
		topics := make([]string, 0)
		for _, subscription := range vt.subscriptions {
			topics = append(topics, subscription.topic)
		}
		tok := vt.client.Unsubscribe(topics...)
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
	if vt.contains(topic) {
		return 0, fmt.Errorf("already subscribed to the %q topic", topic)
	}

	tok := vt.client.Subscribe(topic, byte(qos), vt.messageHandler)
	if tok.Wait() && tok.Error() != nil {
		return 0, fmt.Errorf("subscribe error: %w", tok.Error())
	}
	vt.subscriptions = append(vt.subscriptions, subscription{topic: topic, qos: byte(qos)})
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
	index := v.Int()
	// slices are 0 based
	index--

	if index >= 0 && index < len(vt.subscriptions) {
		subscription := vt.subscriptions[index]
		tok := vt.client.Unsubscribe(subscription.topic)
		if tok.Wait() && tok.Error() != nil {
			return fmt.Errorf("subscribe from %q: %w", subscription.topic, tok.Error())
		}
		vt.subscriptions = slices.Delete(vt.subscriptions, index, index+1)
	}
	return nil
}

func (vt *SubscriberVirtualTable) contains(topic string) bool {
	for _, subscription := range vt.subscriptions {
		if subscription.topic == topic {
			return true
		}
	}
	return false
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
	msg.Ack()
}

func (vt *SubscriberVirtualTable) onConnectionLost(client mqtt.Client, err error) {
	vt.logger.Error("lost connection to the broker", "virtual_table", vt.virtualTableName, "error", err)
}

func (vt *SubscriberVirtualTable) onConnectHandler(client mqtt.Client) {
	vt.logger.Debug("connected to broker", "virtual_table", vt.virtualTableName)
	for _, subscription := range vt.subscriptions {
		client.Subscribe(subscription.topic, subscription.qos, vt.messageHandler)
	}
}

type subscriptionsCursor struct {
	data    []subscription
	current subscription // current row that the cursor points to
	rowid   int64        // current rowid .. negative for EOF
}

func newSubscriptionsCursor(data []subscription) *subscriptionsCursor {
	slices.SortFunc(data, func(a, b subscription) int {
		return cmp.Compare(a.topic, b.topic)
	})
	return &subscriptionsCursor{
		data: data,
	}
}

func (c *subscriptionsCursor) Next() error {
	// EOF
	if c.rowid < 0 || int(c.rowid) >= len(c.data) {
		c.rowid = -1
		return sqlite.SQLITE_OK
	}
	// slices are zero based
	c.current = c.data[c.rowid]
	c.rowid += 1

	return sqlite.SQLITE_OK
}

func (c *subscriptionsCursor) Column(ctx *sqlite.VirtualTableContext, i int) error {
	switch i {
	case 0:
		ctx.ResultText(c.current.topic)
	case 1:
		ctx.ResultInt(int(c.current.qos))
	}
	return nil
}

func (c *subscriptionsCursor) Filter(int, string, ...sqlite.Value) error {
	c.rowid = 0
	return c.Next()
}

func (c *subscriptionsCursor) Rowid() (int64, error) {
	return c.rowid, nil
}

func (c *subscriptionsCursor) Eof() bool {
	return c.rowid < 0
}

func (c *subscriptionsCursor) Close() error {
	return nil
}
