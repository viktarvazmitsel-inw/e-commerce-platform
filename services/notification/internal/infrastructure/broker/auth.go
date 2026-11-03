package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/shop/services/notification/internal/usecase"
)

type authEvent struct {
	Type  string `json:"type"`
	Email string `json:"email"`
	Token string `json:"token"`
}

type notificationPayload struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

func decodeAuthEvent(routingKey string, body []byte) (usecase.CreateNotificationInput, error) {
	if !knownAuthType(routingKey) {
		return usecase.CreateNotificationInput{}, fmt.Errorf("%w: routing key %q", ErrPoison, routingKey)
	}

	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	var event authEvent
	if err := dec.Decode(&event); err != nil {
		return usecase.CreateNotificationInput{}, fmt.Errorf("%w: %v", ErrPoison, err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return usecase.CreateNotificationInput{}, fmt.Errorf("%w: trailing json", ErrPoison)
	}

	if event.Type != routingKey || !knownAuthType(event.Type) {
		return usecase.CreateNotificationInput{}, fmt.Errorf("%w: type %q", ErrPoison, event.Type)
	}
	if strings.TrimSpace(event.Email) == "" || strings.TrimSpace(event.Token) == "" {
		return usecase.CreateNotificationInput{}, fmt.Errorf("%w: email and token are required", ErrPoison)
	}

	payload, err := json.Marshal(notificationPayload{Type: event.Type, Token: event.Token})
	if err != nil {
		return usecase.CreateNotificationInput{}, err
	}
	return usecase.CreateNotificationInput{
		Channel:  "email",
		Receiver: event.Email,
		Payload:  string(payload),
	}, nil
}

func knownAuthType(eventType string) bool {
	switch eventType {
	case authCreatedKey, authPasswordResetKey:
		return true
	default:
		return false
	}
}

func (b *Broker) handleAuth(ctx context.Context, creator NotificationCreator, d amqp.Delivery) error {
	input, err := decodeAuthEvent(d.RoutingKey, d.Body)
	var id string
	if err == nil {
		id, err = creator.Execute(ctx, input)
	}
	return b.settle(ctx, d, emailDLQ, settleAuth(id, err))
}
