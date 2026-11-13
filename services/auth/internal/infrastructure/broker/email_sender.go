package broker

import (
	"authorization/internal/usecase"
	"context"
	"encoding/json/v2"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type MessagePayload struct {
	Type  string         `json:"type"`
	Email string         `json:"email"`
	Token map[string]any `json:"token,omitempty"`
}

type EmailSender struct {
	rmqClient *RMQClient
	exchange  string
	logger    usecase.Logger
}

func NewEmailSender(ctx context.Context, c *RMQClient, l usecase.Logger) (*EmailSender, error) {
	exchangeCh := "auth_exchange"
	if err := c.ch.ExchangeDeclare(
		exchangeCh,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return nil, fmt.Errorf("failed to declare email exchange: %w", err)
	}

	sender := &EmailSender{
		rmqClient: c,
		exchange:  exchangeCh,
		logger:    l,
	}

	returnChan := make(chan amqp.Return, 10)
	c.ch.NotifyReturn(returnChan)

	go sender.handleReturnedMessages(ctx, returnChan)

	return sender, nil
}

func (s *EmailSender) handleReturnedMessages(ctx context.Context, returnChan <-chan amqp.Return) {
	for ret := range returnChan {
		s.logger.Warn(
			ctx,
			"RabbitMQ message returned: unroutable message",
			"exchange", ret.Exchange,
			"routing_key", ret.RoutingKey,
			"reply_code", ret.ReplyCode,
			"reply_text", ret.ReplyText,
		)
	}
}

func (s *EmailSender) SendVerificationEmail(ctx context.Context, email, token string) error {
	payload := MessagePayload{
		Type:  "auth.user.created",
		Email: email,
		Token: map[string]any{
			"verification_token": token,
		},
	}
	return s.publish(ctx, payload, "auth.user.created")
}

func (s *EmailSender) SendPasswordResetEmail(ctx context.Context, email, token string) error {
	payload := MessagePayload{
		Type:  "auth.user.password.reset",
		Email: email,
		Token: map[string]any{
			"password_reset_token": token,
		},
	}
	return s.publish(ctx, payload, "auth.user.password.reset")
}

func (s *EmailSender) publish(ctx context.Context, payload MessagePayload, key string) error {
	message, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	err = s.rmqClient.ch.PublishWithContext(
		ctx,
		s.exchange, // exchange
		key,        // routing key
		true,       // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         message,
			DeliveryMode: amqp.Persistent,
		},
	)
	if err != nil {
		return err
	}
	return nil
}
