package usecase

import (
	"context"

	"github.com/shop/services/notification/internal/domain"
)

type NotificationRepository interface {
	GetNotifications(ctx context.Context, page, quantity int) ([]domain.Notification, int, error)
	GetNotificationByID(ctx context.Context, id string) (*domain.Notification, error)
	CreateNotification(ctx context.Context, notification *domain.Notification) error
	UpdateNotification(ctx context.Context, notification *domain.Notification) error
}

type EmailSender interface {
	Send(ctx context.Context, event, receiver string, payload map[string]any) error
}
