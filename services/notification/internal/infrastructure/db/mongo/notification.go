package mongo

import (
	"time"

	"github.com/shop/services/notification/internal/domain"
)

type notificationDocument struct {
	ID             string    `bson:"_id"`
	Event          string    `bson:"event"`
	Receiver       string    `bson:"receiver"`
	Payload        string    `bson:"payload"`
	Status         string    `bson:"status"`
	ResendAttempts int       `bson:"resend_attempts"`
	CreatedAt      time.Time `bson:"created_at"`
	UpdatedAt      time.Time `bson:"updated_at"`
}

func (d notificationDocument) ToDomain() domain.Notification {
	return domain.Notification{
		ID:             d.ID,
		Event:          d.Event,
		Receiver:       d.Receiver,
		Payload:        d.Payload,
		Status:         domain.Status(d.Status),
		ResendAttempts: d.ResendAttempts,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}
}

func FromDomain(notification *domain.Notification) notificationDocument {
	return notificationDocument{
		ID:             notification.ID,
		Event:          string(notification.Event),
		Receiver:       notification.Receiver,
		Payload:        notification.Payload,
		Status:         string(notification.Status),
		ResendAttempts: notification.ResendAttempts,
		CreatedAt:      notification.CreatedAt.UTC(),
		UpdatedAt:      notification.UpdatedAt.UTC(),
	}
}
