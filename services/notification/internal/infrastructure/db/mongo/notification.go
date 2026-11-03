package mongo

import (
	"time"

	"github.com/shop/services/notification/internal/domain"
)

type notificationDocument struct {
	ID              string    `bson:"_id"`
	Channel         string    `bson:"channel"`
	Receiver        string    `bson:"receiver"`
	Payload         string    `bson:"payload"`
	Status          string    `bson:"status"`
	SendAttempts    int       `bson:"send_attempts"`
	DeliveryVersion int       `bson:"delivery_version"`
	ClaimedUntil    time.Time `bson:"claimed_until"`
	AwaitingPublish bool      `bson:"awaiting_publish"`
	CreatedAt       time.Time `bson:"created_at"`
	UpdatedAt       time.Time `bson:"updated_at"`
}

func (d notificationDocument) ToDomain() domain.Notification {
	return domain.Notification{
		ID:              d.ID,
		Channel:         d.Channel,
		Receiver:        d.Receiver,
		Payload:         d.Payload,
		Status:          domain.Status(d.Status),
		SendAttempts:    d.SendAttempts,
		DeliveryVersion: d.DeliveryVersion,
		ClaimedUntil:    d.ClaimedUntil,
		AwaitingPublish: d.AwaitingPublish,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}
}

func FromDomain(notification *domain.Notification) notificationDocument {
	return notificationDocument{
		ID:              notification.ID,
		Channel:         notification.Channel,
		Receiver:        notification.Receiver,
		Payload:         notification.Payload,
		Status:          string(notification.Status),
		SendAttempts:    notification.SendAttempts,
		DeliveryVersion: notification.DeliveryVersion,
		ClaimedUntil:    notification.ClaimedUntil.UTC(),
		AwaitingPublish: notification.AwaitingPublish,
		CreatedAt:       notification.CreatedAt.UTC(),
		UpdatedAt:       notification.UpdatedAt.UTC(),
	}
}
