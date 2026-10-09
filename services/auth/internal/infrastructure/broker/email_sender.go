package broker

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type MessagePayload struct {
	Type  string `json:"type"`
	Email string `json:"email"`
	Token string `json:"token"`
}

type EmailSender struct {
	rmqClient *RMQClient
	exchange  string
}

func NewEmailSender(rmqClient *RMQClient) (*EmailSender, error) {
	exchangeCh := "email_exchange"
	if err := rmqClient.ch.ExchangeDeclare(
		exchangeCh,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return nil, fmt.Errorf("failed to declare email exchange: %w", err)
	}

	return &EmailSender{
		rmqClient: rmqClient,
		exchange:  exchangeCh,
	}, nil
}

func (s *EmailSender) SendVerificationEmail(ctx context.Context, email, token string) error {
	payload := MessagePayload{
		Type:  "verification",
		Email: email,
		Token: token,
	}
	return s.publish(ctx, payload, "verification")
}

func (s *EmailSender) SendPasswordResetEmail(ctx context.Context, email, token string) error {
	payload := MessagePayload{
		Type:  "password_reset",
		Email: email,
		Token: token,
	}
	return s.publish(ctx, payload, "password_reset")
}

func (s *EmailSender) publish(ctx context.Context, payload MessagePayload, key string) error {
	message, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	s.rmqClient.mu.Lock()
	defer s.rmqClient.mu.Unlock()

	if s.rmqClient.ch == nil {
		return fmt.Errorf("no channel to publish")
	}

	err = s.rmqClient.ch.PublishWithContext(
		ctx,
		s.exchange, // exchange
		key,        // routing key
		false,      // mandatory
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
