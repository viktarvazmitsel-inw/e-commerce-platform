package usecase

import (
	"context"
	"time"

	"github.com/shop/services/notification/internal/domain"
)

type NotificationRepository interface {
	GetNotifications(ctx context.Context, page, quantity int) ([]domain.Notification, int, error)
	GetNotificationsByStatuses(ctx context.Context, statuses []domain.Status, retryWindow time.Duration) ([]domain.Notification, error)
	GetNotificationByID(ctx context.Context, id string) (*domain.Notification, error)
	CreateNotification(ctx context.Context, notification *domain.Notification) error
	UpdateNotification(ctx context.Context, notification *domain.Notification) error
}

type MessageQueuePublisher interface {
	PublishForDelivery(ctx context.Context, notificationID string) error
}

type EmailSender interface {
	Send(ctx context.Context, channel, receiver, payload string) error
}
