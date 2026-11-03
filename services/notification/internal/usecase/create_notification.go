package usecase

import (
	"context"
	"fmt"

	"github.com/shop/services/notification/internal/domain"
)

type CreateNotificationInput struct {
	Channel  string
	Receiver string
	Payload  string
}

type CreateNotificationUseCase struct {
	repo      NotificationRepository
	publisher MessageQueuePublisher
}

func NewCreateNotificationUseCase(r NotificationRepository, p MessageQueuePublisher) *CreateNotificationUseCase {
	return &CreateNotificationUseCase{repo: r, publisher: p}
}

func (uc *CreateNotificationUseCase) Execute(ctx context.Context, input CreateNotificationInput) (string, error) {
	notification, err := domain.NewNotification(input.Channel, input.Receiver, input.Payload)
	if err != nil {
		return "", err
	}

	if err := uc.repo.CreateNotification(ctx, notification); err != nil {
		return "", fmt.Errorf("create notification: %w", err)
	}

	if err := uc.publisher.PublishForDelivery(ctx, notification.ID, notification.DeliveryVersion); err != nil {
		return notification.ID, fmt.Errorf("publish notification for delivery: %w", err)
	}
	if err := uc.repo.MarkDeliveryPublished(ctx, notification.ID, notification.DeliveryVersion); err != nil {
		return notification.ID, fmt.Errorf("update notification: %w", err)
	}

	return notification.ID, nil
}
