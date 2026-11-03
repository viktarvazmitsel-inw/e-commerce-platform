package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/shop/services/notification/internal/domain"
)

type ResendNotificationsUseCase struct {
	repo        NotificationRepository
	publisher   MessageQueuePublisher
	retryWindow time.Duration
}

func NewResendNotificationsUseCase(r NotificationRepository, p MessageQueuePublisher, retryWindow time.Duration) *ResendNotificationsUseCase {
	return &ResendNotificationsUseCase{repo: r, publisher: p, retryWindow: retryWindow}
}

func (uc *ResendNotificationsUseCase) Execute(ctx context.Context) (int, error) {
	notifications, err := uc.repo.GetNotificationsByStatuses(ctx, []domain.Status{
		domain.StatusPending,
		domain.StatusFailed,
	}, uc.retryWindow)
	if err != nil {
		return 0, fmt.Errorf("load notifications: %w", err)
	}

	now := time.Now().UTC()
	cutoff := now.Add(-uc.retryWindow)
	queued := 0
	for i := range notifications {
		notification := &notifications[i]
		switch notification.Status {
		case domain.StatusFailed:
			if err := notification.ResetForResend(); err != nil {
				return queued, err
			}
			if err := uc.repo.UpdateNotification(ctx, notification); err != nil {
				return queued, fmt.Errorf("update notification: %w", err)
			}
		case domain.StatusPending:
			if notification.ClaimedUntil.After(now) {
				continue
			}
			if !notification.AwaitingPublish && !notification.UpdatedAt.Before(cutoff) {
				continue
			}
			if !notification.AwaitingPublish {
				if err := notification.SupersedeDelivery(); err != nil {
					return queued, err
				}
				if err := uc.repo.UpdateNotification(ctx, notification); err != nil {
					return queued, fmt.Errorf("update notification: %w", err)
				}
			}
		default:
			continue
		}

		if err := uc.publisher.PublishForDelivery(ctx, notification.ID, notification.DeliveryVersion); err != nil {
			return queued, fmt.Errorf("publish notification for delivery: %w", err)
		}
		if err := uc.repo.MarkDeliveryPublished(ctx, notification.ID, notification.DeliveryVersion); err != nil {
			return queued, fmt.Errorf("update notification: %w", err)
		}
		queued++
	}

	return queued, nil
}
