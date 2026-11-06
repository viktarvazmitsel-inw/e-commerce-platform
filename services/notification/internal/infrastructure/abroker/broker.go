package abroker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	internalExchange = "internal_exchange"
	retryExchange    = "retry_exchange"
	deadLetterQueue  = "dead_letter_queue"
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
	conn      *amqp.Connection
	publishCh *amqp.Channel
	publishMu sync.Mutex
	returns   chan amqp.Return
	wg        sync.WaitGroup
}

func NewConnection(ctx context.Context, dsn *RMQConfig, router *BrokerRouter) (*Broker, error) {
	if err := validateRouter(router); err != nil {
		return nil, err
	}

	conn, err := amqp.Dial(dsn.DSN())
	if err != nil {
		return nil, err
	}

	b := &Broker{
		conn: conn,
	}

	if err := b.open(router); err != nil {
		conn.Close()
		return nil, err
	}

	return b, nil
}

func (b *Broker) open(router *BrokerRouter) error {
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

	if err := b.declareBrokerTopology(b.publishCh, router.Services); err != nil {
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

func (b *Broker) Serve(ctx context.Context, router *BrokerRouter) error {
	if err := validateRouter(router); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, len(router.Services))
	for _, service := range router.Services {
		qName := service + "_queue"
		if err := b.subscribe(ctx, qName, 10, router, errCh); err != nil {
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
	maxQueue int,
	router *BrokerRouter,
	errCh chan<- error,
) error {
	ch, err := b.conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	if err := ch.Qos(maxQueue, 0, false); err != nil {
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
			b.handleDelivery(ctx, msg, router)
		}
	}
}

func (b *Broker) handleDelivery(ctx context.Context, msg amqp.Delivery, router *BrokerRouter) {
	event := msg.RoutingKey
	handler, ok := router.HandlerMap[event]
	if !ok {
		log.Printf("no handler found for event: %s", event)
		msg.Nack(false, false)
		return
	}

	retryCount := getRetryCount(msg.Headers)

	if retryCount >= 3 {
		if err := router.DeadLetterHandler(ctx, msg.Body, retryCount); err != nil {
			log.Printf("failed to handle dead letter: %v", err)
			msg.Nack(false, false)
			return
		}

		msg.Ack(false)
		return
	}

	if err := handler(ctx, msg.Body, retryCount); err != nil {
		retryCount++
		err := b.resendMessage(ctx, msg.Body, retryCount, msg.RoutingKey, msg.CorrelationId, msg.MessageId)
		if err != nil {
			msg.Nack(false, false)
			log.Printf("failed to resend message: %v", err)
			return
		}
	}
	msg.Ack(false)
}

func getRetryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}
	retryCount, ok := headers["x-retry-count"].(int64)
	if !ok {
		return 0
	}
	return int(retryCount)
}

func (b *Broker) resendMessage(ctx context.Context, body []byte, retryCount int, routingKey, correlationID, messageID string) error {
	headers := amqp.Table{
		"x-routing-key": routingKey,
		"x-retry-count": int64(retryCount),
	}
	publishing := amqp.Publishing{
		ContentType:   "application/json",
		Body:          body,
		DeliveryMode:  amqp.Persistent,
		CorrelationId: correlationID,
		MessageId:     messageID,
		Headers:       headers,
	}

	retryInterval := selectRetryInterval(retryCount)
	if retryInterval == "" {
		return b.publish(ctx, "", deadLetterQueue, publishing)
	}

	headers["x-retry-interval"] = retryInterval
	return b.publish(ctx, retryExchange, routingKey, publishing)
}

func (b *Broker) publish(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
	b.publishMu.Lock()
	defer b.publishMu.Unlock()

	confirmation, err := b.publishCh.PublishWithDeferredConfirmWithContext(ctx, exchange, routingKey, true, false, msg)
	if err != nil {
		return err
	}
	if confirmation == nil {
		return errors.New("publish channel is not confirming")
	}

	// RabbitMQ sends basic.return before basic.ack, so a return for this
	// publish is already buffered when the confirmation completes.
	confirmation.Wait()
	if ret, ok := b.returnedPublish(); ok {
		return fmt.Errorf("unroutable publish to exchange %q with routing key %q: %s", ret.Exchange, ret.RoutingKey, ret.ReplyText)
	}
	if !confirmation.Acked() {
		return errors.New("broker nacked publish")
	}
	return nil
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

func (b *Broker) Close() error {
	b.wg.Wait()
	if err := b.publishCh.Close(); err != nil {
		return fmt.Errorf("close publish channel: %w", err)
	}
	if err := b.conn.Close(); err != nil {
		return fmt.Errorf("close connection: %w", err)
	}
	return nil
}
