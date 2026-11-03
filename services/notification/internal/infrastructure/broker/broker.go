package broker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/shop/services/notification/internal/usecase"
)

const (
	emailExchange        = "email_exchange"
	emailQueue           = "notification.email"
	emailDLQ             = "notification.email.dlq"
	deliveryQueue        = "notification.delivery"
	deliveryDLQ          = "notification.delivery.dlq"
	deadLetterExchange   = "notification.dlx"
	prefetch             = 10
	authCreatedKey       = "user.created"
	authPasswordResetKey = "user.password_reset"
)

var ErrPoison = errors.New("poison broker message")

type NotificationCreator interface {
	Execute(ctx context.Context, input usecase.CreateNotificationInput) (string, error)
}

type NotificationSender interface {
	Execute(ctx context.Context, input usecase.SendNotificationInput) error
}

type Broker struct {
	conn       *amqp.Connection
	publishCh  *amqp.Channel
	authCh     *amqp.Channel
	deliveryCh *amqp.Channel

	mu        sync.Mutex
	publishMu sync.Mutex
	running   bool
	closeOnce sync.Once
}

func New(url string) (*Broker, error) {
	if strings.TrimSpace(url) == "" {
		return nil, errors.New("broker url is empty")
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("dial broker: %w", err)
	}

	b := &Broker{conn: conn}
	if err := b.open(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return b, nil
}

func (b *Broker) open() error {
	var err error
	b.publishCh, err = b.conn.Channel()
	if err != nil {
		return fmt.Errorf("open publish channel: %w", err)
	}
	if err := declareTopology(b.publishCh); err != nil {
		return err
	}

	b.authCh, err = b.conn.Channel()
	if err != nil {
		return fmt.Errorf("open auth channel: %w", err)
	}
	if err := b.authCh.Qos(prefetch, 0, false); err != nil {
		return fmt.Errorf("set auth prefetch: %w", err)
	}

	b.deliveryCh, err = b.conn.Channel()
	if err != nil {
		return fmt.Errorf("open delivery channel: %w", err)
	}
	if err := b.deliveryCh.Qos(prefetch, 0, false); err != nil {
		return fmt.Errorf("set delivery prefetch: %w", err)
	}
	return nil
}

func declareTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(
		emailExchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare email exchange: %w", err)
	}
	if err := ch.ExchangeDeclare(
		deadLetterExchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare dead letter exchange: %w", err)
	}

	if err := declareQueue(ch, emailDLQ, nil); err != nil {
		return err
	}
	if err := declareQueue(ch, deliveryDLQ, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(emailDLQ, emailQueue, deadLetterExchange, false, nil); err != nil {
		return fmt.Errorf("bind email dead letter queue: %w", err)
	}
	if err := ch.QueueBind(deliveryDLQ, deliveryQueue, deadLetterExchange, false, nil); err != nil {
		return fmt.Errorf("bind delivery dead letter queue: %w", err)
	}

	dlx := amqp.Table{
		"x-dead-letter-exchange":    deadLetterExchange,
		"x-dead-letter-routing-key": emailQueue,
	}
	if err := declareQueue(ch, emailQueue, dlx); err != nil {
		return err
	}
	for _, key := range []string{authCreatedKey, authPasswordResetKey} {
		if err := ch.QueueBind(emailQueue, key, emailExchange, false, nil); err != nil {
			return fmt.Errorf("bind %s: %w", key, err)
		}
	}

	deliveryDLX := amqp.Table{
		"x-dead-letter-exchange":    deadLetterExchange,
		"x-dead-letter-routing-key": deliveryQueue,
	}
	if err := declareQueue(ch, deliveryQueue, deliveryDLX); err != nil {
		return err
	}
	return nil
}

func declareQueue(ch *amqp.Channel, name string, args amqp.Table) error {
	if args == nil {
		args = amqp.Table{}
	}
	args[amqp.QueueTypeArg] = amqp.QueueTypeQuorum
	_, err := ch.QueueDeclare(name, true, false, false, false, args)
	if err != nil {
		return fmt.Errorf("declare queue %s: %w", name, err)
	}
	return nil
}

func (b *Broker) Run(ctx context.Context, creator NotificationCreator, sender NotificationSender) error {
	if creator == nil || sender == nil {
		return errors.New("broker creator and sender are required")
	}

	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return errors.New("broker is already running")
	}
	if b.publishCh == nil {
		b.mu.Unlock()
		return errors.New("broker is closed")
	}
	b.running = true
	b.mu.Unlock()

	authMsgs, err := b.authCh.Consume(emailQueue, "", false, false, false, false, nil)
	if err != nil {
		b.stopRunning()
		return fmt.Errorf("consume auth events: %w", err)
	}
	deliveryMsgs, err := b.deliveryCh.Consume(deliveryQueue, "", false, false, false, false, nil)
	if err != nil {
		b.stopRunning()
		return fmt.Errorf("consume deliveries: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		errCh <- b.loop(ctx, authMsgs, func(d amqp.Delivery) error {
			return b.handleAuth(ctx, creator, d)
		})
	}()
	go func() {
		defer wg.Done()
		errCh <- b.loop(ctx, deliveryMsgs, func(d amqp.Delivery) error {
			return b.handleDelivery(ctx, sender, d)
		})
	}()

	var runErr error
	select {
	case <-ctx.Done():
		runErr = ctx.Err()
	case runErr = <-errCh:
		cancel()
	}
	b.Close()
	wg.Wait()
	return runErr
}

func (b *Broker) stopRunning() {
	b.mu.Lock()
	b.running = false
	b.mu.Unlock()
}

func (b *Broker) loop(ctx context.Context, msgs <-chan amqp.Delivery, handle func(amqp.Delivery) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-msgs:
			if !ok {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return errors.New("broker deliveries closed")
			}
			if err := handle(d); err != nil {
				return err
			}
		}
	}
}

func (b *Broker) publish(ctx context.Context, queue string, body []byte) error {
	b.publishMu.Lock()
	defer b.publishMu.Unlock()

	b.mu.Lock()
	ch := b.publishCh
	b.mu.Unlock()
	if ch == nil {
		return errors.New("broker is closed")
	}
	return ch.PublishWithContext(ctx, "", queue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (b *Broker) settle(ctx context.Context, d amqp.Delivery, dlq string, action disposition) error {
	switch action {
	case ack:
		return d.Ack(false)
	case requeue:
		return d.Nack(false, true)
	case deadLetter:
		if err := b.publish(ctx, dlq, d.Body); err != nil {
			_ = d.Nack(false, true)
			return err
		}
		return d.Ack(false)
	default:
		return d.Nack(false, true)
	}
}

func (b *Broker) Close() {
	b.closeOnce.Do(func() {
		b.mu.Lock()
		b.publishCh = nil
		b.running = false
		conn := b.conn
		b.conn = nil
		b.mu.Unlock()

		if conn != nil {
			_ = conn.Close()
		}
	})
}

var (
	_ NotificationCreator           = (*usecase.CreateNotificationUseCase)(nil)
	_ NotificationSender            = (*usecase.SendNotificationUseCase)(nil)
	_ usecase.MessageQueuePublisher = (*Broker)(nil)
)
