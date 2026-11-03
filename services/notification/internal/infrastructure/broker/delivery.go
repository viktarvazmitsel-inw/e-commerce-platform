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

type deliveryMessage struct {
	NotificationID  string `json:"notification_id"`
	DeliveryVersion int    `json:"delivery_version"`
}

func (b *Broker) PublishForDelivery(ctx context.Context, notificationID string, deliveryVersion int) error {
	body, err := encodeDelivery(notificationID, deliveryVersion)
	if err != nil {
		return err
	}
	if err := b.publish(ctx, deliveryQueue, body); err != nil {
		return fmt.Errorf("publish notification for delivery: %w", err)
	}
	return nil
}

func encodeDelivery(notificationID string, deliveryVersion int) ([]byte, error) {
	if strings.TrimSpace(notificationID) == "" || deliveryVersion < 1 {
		return nil, fmt.Errorf("invalid delivery message")
	}
	body, err := json.Marshal(deliveryMessage{
		NotificationID:  notificationID,
		DeliveryVersion: deliveryVersion,
	})
	if err != nil {
		return nil, fmt.Errorf("encode delivery message: %w", err)
	}
	return body, nil
}

func decodeDelivery(body []byte) (deliveryMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	var message deliveryMessage
	if err := dec.Decode(&message); err != nil {
		return deliveryMessage{}, fmt.Errorf("%w: %v", ErrPoison, err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return deliveryMessage{}, fmt.Errorf("%w: trailing json", ErrPoison)
	}
	if strings.TrimSpace(message.NotificationID) == "" || message.DeliveryVersion < 1 {
		return deliveryMessage{}, fmt.Errorf("%w: notification id and delivery version are required", ErrPoison)
	}
	return message, nil
}

func (b *Broker) handleDelivery(ctx context.Context, sender NotificationSender, d amqp.Delivery) error {
	message, err := decodeDelivery(d.Body)
	if err == nil {
		err = sender.Execute(ctx, usecase.SendNotificationInput{
			NotificationID:  message.NotificationID,
			DeliveryVersion: message.DeliveryVersion,
		})
	}
	return b.settle(ctx, d, deliveryDLQ, settleDelivery(err))
}
