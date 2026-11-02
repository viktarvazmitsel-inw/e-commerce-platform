package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/shop/services/notification/internal/domain"
)

type SendNotificationInput struct {
	NotificationID string
	SendAttempt    int
}

type SendNotificationUseCase struct {
	repo           NotificationRepository
	sender         EmailSender
	maxSendAttempt int
}

func NewSendNotificationUseCase(r NotificationRepository, s EmailSender, maxSendAttempt int) *SendNotificationUseCase {
	return &SendNotificationUseCase{repo: r, sender: s, maxSendAttempt: maxSendAttempt}
}

func (uc *SendNotificationUseCase) Execute(ctx context.Context, input SendNotificationInput) (int, error) {
	if input.SendAttempt > uc.maxSendAttempt {
		return uc.failForAttemptLimit(ctx, input)
	}

	notification, err := uc.load(ctx, input.NotificationID)
	if err != nil {
		return input.SendAttempt, err
	}
	if notification.Status != domain.StatusPending {
		return input.SendAttempt, nil
	}

	if err := uc.sender.Send(ctx, notification.Channel, notification.Receiver, notification.Payload); err != nil {
		next := input.SendAttempt + 1
		if setErr := notification.SetSendAttempts(next); setErr != nil {
			return input.SendAttempt, setErr
		}
		_ = uc.repo.UpdateNotification(ctx, notification)
		return next, fmt.Errorf("%w: %w", ErrSendFailed, err)
	}

	if err := notification.SetSendAttempts(input.SendAttempt); err != nil {
		return input.SendAttempt, err
	}
	if err := notification.SetStatus(domain.StatusSent); err != nil {
		return input.SendAttempt, err
	}
	if err := uc.repo.UpdateNotification(ctx, notification); err != nil {
		return input.SendAttempt, fmt.Errorf("update notification: %w", err)
	}
	return input.SendAttempt, nil
}

func (uc *SendNotificationUseCase) failForAttemptLimit(ctx context.Context, input SendNotificationInput) (int, error) {
	notification, err := uc.load(ctx, input.NotificationID)
	if err != nil {
		return input.SendAttempt, err
	}

	if notification.Status == domain.StatusPending {
		if err := notification.SetSendAttempts(input.SendAttempt); err != nil {
			return input.SendAttempt, err
		}
		if err := notification.SetStatus(domain.StatusFailed); err != nil {
			return input.SendAttempt, err
		}
		if err := uc.repo.UpdateNotification(ctx, notification); err != nil {
			return input.SendAttempt, fmt.Errorf("update notification: %w", err)
		}
	}

	return input.SendAttempt, ErrSendAttemptLimit
}

func (uc *SendNotificationUseCase) load(ctx context.Context, id string) (*domain.Notification, error) {
	notification, err := uc.repo.GetNotificationByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			return nil, ErrNotificationNotFound
		}
		return nil, fmt.Errorf("load notification: %w", err)
	}
	if notification == nil {
		return nil, ErrNotificationNotFound
	}
	return notification, nil
}
