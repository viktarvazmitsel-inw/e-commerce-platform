package broker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	internalExchange      = "internal_exchange"
	retryExchange         = "retry_exchange"
	deadLetterQueue       = "dead_letter_queue"
	originalIDHeader      = "x-original-id"
	publishConfirmTimeout = 30 * time.Second
)

type RMQConfig struct {
	Host     string
	Port     string
	User     string
	Password string
}

func (c *RMQConfig) DSN() string {
	u := &url.URL{
		Scheme: "amqp",
		User:   url.UserPassword(c.User, c.Password),
		Host:   fmt.Sprintf("%s:%s", c.Host, c.Port),
	}

	return u.String()
}

type Broker struct {
	conn       *amqp.Connection
	publishCh  *amqp.Channel
	publishMu  sync.Mutex
	publishErr error
	returns    chan amqp.Return
	wg         sync.WaitGroup

	cancelMu sync.Mutex
	cancel   context.CancelFunc
}

func NewConnection(ctx context.Context, dsn *RMQConfig, services *BrokerServiceList) (*Broker, error) {
	if len(services.List) == 0 {
		return nil, errors.New("services list is required")
	}

	conn, err := amqp.Dial(dsn.DSN())
	if err != nil {
		return nil, err
	}

	b := &Broker{
		conn: conn,
	}

	if err := b.open(services.List); err != nil {
		conn.Close()
		return nil, err
	}

	return b, nil
}

func (b *Broker) open(services []string) error {
	var err error
	b.publishCh, err = b.conn.Channel()
	if err != nil {
		return fmt.Errorf("open publish channel: %w", err)
	}
	b.returns = make(chan amqp.Return, 1)
	b.publishCh.NotifyReturn(b.returns)
	if err := b.publishCh.Confirm(false); err != nil {
		return fmt.Errorf("enable publish confirms: %w", err)
	}

	if err := b.declareBrokerTopology(b.publishCh, services); err != nil {
		return fmt.Errorf("declare topology: %w", err)
	}

	return nil
}

func (b *Broker) declareBrokerTopology(ch *amqp.Channel, services []string) error {
	for _, s := range services {
		exchangeName := s + "_exchange"
		if err := ch.ExchangeDeclare(
			exchangeName,
			"topic",
			true,
			false,
			false,
			false,
			nil,
		); err != nil {
			return fmt.Errorf("declare %s exchange: %w", exchangeName, err)
		}
	}

	if err := ch.ExchangeDeclare(
		internalExchange,
		"topic",
		true,
		false,
		true,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare internal exchange: %w", err)
	}

	if err := ch.ExchangeDeclare(
		retryExchange,
		"headers",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare dead letter exchange: %w", err)
	}

	type retryQueueConfig struct {
		qName         string
		ttl           int32
		retryInterval string
	}

	queueConfigs := []retryQueueConfig{
		{
			qName:         "retry_10",
			ttl:           10000,
			retryInterval: "10s",
		},
		{
			qName:         "retry_60",
			ttl:           60000,
			retryInterval: "60s",
		},
		{
			qName:         "retry_600",
			ttl:           600000,
			retryInterval: "600s",
		},
	}

	for _, c := range queueConfigs {
		_, err := ch.QueueDeclare(
			c.qName,
			true,
			false,
			false,
			false,
			amqp.Table{
				"x-message-ttl":          c.ttl,
				"x-dead-letter-exchange": internalExchange,
			})
		if err != nil {
			return fmt.Errorf("declare resend queue: %w", err)
		}

		if err := ch.QueueBind(
			c.qName,
			"",
			retryExchange,
			false,
			amqp.Table{
				"x-match":          "all",
				"x-retry-interval": c.retryInterval,
			},
		); err != nil {
			return fmt.Errorf("bind %s queue: %w", c.qName, err)
		}
	}

	for _, s := range services {
		qName := s + "_queue"
		pattern := s + ".#"
		exchanger := s + "_exchange"
		if _, err := ch.QueueDeclare(
			qName,
			true,
			false,
			false,
			false,
			nil,
		); err != nil {
			return fmt.Errorf("declare %s queue: %w", qName, err)
		}

		if err := ch.QueueBind(
			qName,
			pattern,
			exchanger,
			false,
			nil,
		); err != nil {
			return fmt.Errorf("bind %s queue: %w", qName, err)
		}

		if err := ch.QueueBind(
			qName,
			pattern,
			internalExchange,
			false,
			nil,
		); err != nil {
			return fmt.Errorf("bind %s queue to internal exchange: %w", qName, err)
		}
	}

	if _, err := ch.QueueDeclare(
		deadLetterQueue,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare dead letter queue: %w", err)
	}

	return nil
}

func (b *Broker) Serve(ctx context.Context, router *BrokerRouter, services *BrokerServiceList) error {
	ctx, cancel := context.WithCancel(ctx)
	b.setCancel(cancel)
	defer cancel()

	errCh := make(chan error, len(services.List))
	for _, service := range services.List {
		if err := services.ValidateService(service); err != nil {
			return fmt.Errorf("validate service %s: %w", service, err)
		}

		qName := service + "_queue"
		if err := b.subscribe(ctx, qName, router, errCh); err != nil {
			cancel()
			b.wg.Wait()
			return fmt.Errorf("subscribe to %s queue: %w", qName, err)
		}
	}

	var runErr error
	select {
	case <-ctx.Done():
		runErr = ctx.Err()
	case runErr = <-errCh:
		cancel()
	}
	b.wg.Wait()
	return runErr
}

func (b *Broker) subscribe(
	ctx context.Context,
	queueName string,
	router *BrokerRouter,
	errCh chan<- error,
) error {
	ch, err := b.conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	if err := ch.Qos(10, 0, false); err != nil {
		_ = ch.Close()
		return fmt.Errorf("set prefetch: %w", err)
	}

	msgs, err := ch.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("consume messages: %w", err)
	}

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		defer ch.Close()
		errCh <- b.consume(ctx, msgs, router)
	}()

	return nil
}

func (b *Broker) consume(ctx context.Context, msgs <-chan amqp.Delivery, router *BrokerRouter) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-msgs:
			if !ok {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return errors.New("broker deliveries closed")
			}

			var metadata MessageMetadata

			originalID, ok := msg.Headers["x-original-id"].(string)
			if !ok {
				originalID = ""
			}

			retryCount := retryCountFromHeader(msg.Headers)

			metadata = MessageMetadata{
				OriginalID: originalID,
				RetryCount: retryCount,
				RoutingKey: msg.RoutingKey,
			}

			b.handleDelivery(ctx, msg, metadata, router)
		}
	}
}

func (b *Broker) handleDelivery(
	ctx context.Context,
	msg amqp.Delivery,
	metadata MessageMetadata,
	router *BrokerRouter,
) {
	event := msg.RoutingKey
	handler, ok := router.HandlerMap[event]
	if !ok {
		log.Printf("no handler found for event: %s", event)
		msg.Nack(false, false)
		return
	}

	handler(ctx, msg.Body, metadata)
	msg.Ack(false)
}

func (b *Broker) ResendMessage(
	ctx context.Context,
	body []byte,
	metadata MessageMetadata,
) error {

	headers := amqp.Table{
		"x-retry-count": int32(metadata.RetryCount),
		"x-original-id": metadata.OriginalID,
	}
	publishing := amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
		Headers:      headers,
	}

	retryInterval := selectRetryInterval(metadata.RetryCount)
	if retryInterval == "" {
		return b.publish(ctx, "", deadLetterQueue, publishing)
	}

	headers["x-retry-interval"] = retryInterval
	return b.publish(ctx, retryExchange, metadata.RoutingKey, publishing)
}

func (b *Broker) publish(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
	b.publishMu.Lock()
	defer b.publishMu.Unlock()

	if b.publishErr != nil {
		return b.publishErr
	}

	confirmation, err := b.publishCh.PublishWithDeferredConfirmWithContext(
		ctx,
		exchange,
		routingKey,
		true,
		false,
		msg,
	)
	if err != nil {
		return err
	}
	if confirmation == nil {
		return errors.New("publish channel is not confirming")
	}

	// RabbitMQ sends basic.return before basic.ack, so a return for this
	// publish is already buffered when the confirmation completes.
	waitCtx, waitCancel := context.WithTimeout(ctx, publishConfirmTimeout)
	defer waitCancel()
	acked, err := confirmation.WaitContext(waitCtx)
	if err != nil {
		// A late basic.return can still arrive on this channel. Leave it
		// unread and refuse later publishes so it cannot be charged to them.
		b.retirePublishChannel(err)
		return b.publishErr
	}
	if ret, ok := b.returnedPublish(); ok {
		return fmt.Errorf("unroutable publish to exchange %q with routing key %q: %s", ret.Exchange, ret.RoutingKey, ret.ReplyText)
	}
	if !acked {
		return errors.New("broker nacked publish")
	}
	return nil
}

// retirePublishChannel drops the publish channel after a confirm wait was
// interrupted. Caller holds publishMu. The channel is left for connection
// close to release; closing it here would wait on the broker again.
func (b *Broker) retirePublishChannel(cause error) {
	if b.publishErr == nil {
		b.publishErr = fmt.Errorf("publish confirm interrupted: %w", cause)
	}
	b.publishCh = nil
}

func (b *Broker) returnedPublish() (amqp.Return, bool) {
	select {
	case ret, ok := <-b.returns:
		if !ok {
			return amqp.Return{}, false
		}
		return ret, true
	default:
		return amqp.Return{}, false
	}
}

// retryCountFromHeader reads x-retry-count as this client returns it.
// A published Go int is encoded as AMQP int32 and decoded as int32, so an
// assert to int always fails and every resend looks like the first attempt.
func retryCountFromHeader(headers amqp.Table) int {
	switch n := headers["x-retry-count"].(type) {
	case int32:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}

func selectRetryInterval(retryCount int) string {
	switch retryCount {
	case 1:
		return "10s"
	case 2:
		return "60s"
	case 3:
		return "600s"
	default:
		return ""
	}
}

func (b *Broker) setCancel(cancel context.CancelFunc) {
	b.cancelMu.Lock()
	b.cancel = cancel
	b.cancelMu.Unlock()
}

func (b *Broker) stopConsuming() {
	b.cancelMu.Lock()
	cancel := b.cancel
	b.cancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (b *Broker) Close() error {
	// Cancel so an in-flight confirm wait returns, then close the connection
	// so delivery channels unblock, then wait for the consumers to leave.
	b.stopConsuming()

	var err error
	if b.conn != nil {
		err = b.conn.Close()
	}
	b.wg.Wait()
	if err != nil {
		return fmt.Errorf("close connection: %w", err)
	}
	return nil
}
