package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/shop/services/notification/internal/domain"
)

type SendNotificationInput struct {
	NotificationID string
	Event          string
	Receiver       string
	Payload        map[string]any
}

type SendNotificationOutput struct {
	ResendCount    int
	NotificationID string
}

type SendNotificationUseCase struct {
	repo   NotificationRepository
	sender EmailSender
}

func NewSendNotificationUseCase(r NotificationRepository, s EmailSender) *SendNotificationUseCase {
	return &SendNotificationUseCase{
		repo:   r,
		sender: s,
	}
}

func (uc *SendNotificationUseCase) Execute(
	ctx context.Context,
	input SendNotificationInput,
) (SendNotificationOutput, error) {
	var notification *domain.Notification
	var err error
	if input.NotificationID == "" {
		notification, err = domain.NewNotification(input.Event, input.Receiver, input.Payload)
		if err != nil {
			return SendNotificationOutput{}, err
		}
		if err := uc.repo.CreateNotification(ctx, notification); err != nil {
			return SendNotificationOutput{}, fmt.Errorf("create notification: %w", err)
		}
	} else {
		notification, err = uc.load(ctx, input.NotificationID)
		if err != nil {
			return SendNotificationOutput{}, err
		}
		if notification.Status == domain.StatusSent || notification.Status == domain.StatusFailed {
			return SendNotificationOutput{
				NotificationID: notification.ID,
			}, nil
		}
		if notification.ResendAttempts >= 3 {
			if err := notification.SetStatus(domain.StatusFailed); err != nil {
				return SendNotificationOutput{
					NotificationID: notification.ID,
				}, err
			}
			if err := uc.repo.UpdateNotification(ctx, notification); err != nil {
				return SendNotificationOutput{
					NotificationID: notification.ID,
				}, fmt.Errorf("update notification: %w", err)
			}
			return SendNotificationOutput{
				ResendCount:    notification.ResendAttempts,
				NotificationID: notification.ID,
			}, ErrMaxResendAttemptsReached
		}
	}

	if err := uc.sender.Send(ctx, notification.Event, notification.Receiver, notification.Payload); err != nil {
		if err := notification.IncreaseSendAttempts(); err != nil {
			return SendNotificationOutput{
				NotificationID: notification.ID,
			}, err
		}
		if err := uc.repo.UpdateNotification(ctx, notification); err != nil {
			return SendNotificationOutput{
				NotificationID: notification.ID,
			}, fmt.Errorf("update notification: %w", err)
		}
		return SendNotificationOutput{
			ResendCount:    notification.ResendAttempts,
			NotificationID: notification.ID,
		}, err
	}

	if err := notification.SetStatus(domain.StatusSent); err != nil {
		return SendNotificationOutput{
			NotificationID: notification.ID,
		}, err
	}

	if err := uc.repo.UpdateNotification(ctx, notification); err != nil {
		return SendNotificationOutput{
			NotificationID: notification.ID,
		}, fmt.Errorf("update notification: %w", err)
	}

	return SendNotificationOutput{
		NotificationID: notification.ID,
	}, nil
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
