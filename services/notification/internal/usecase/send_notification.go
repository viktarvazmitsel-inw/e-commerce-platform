package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shop/services/notification/internal/domain"
)

type SendNotificationInput struct {
	NotificationID  string
	DeliveryVersion int
}

type SendNotificationUseCase struct {
	repo           NotificationRepository
	sender         EmailSender
	maxSendAttempt int
	claimLease     time.Duration
}

func NewSendNotificationUseCase(r NotificationRepository, s EmailSender, maxSendAttempt int, claimLease time.Duration) (*SendNotificationUseCase, error) {
	if maxSendAttempt < 1 {
		return nil, fmt.Errorf("max send attempt must be positive")
	}
	if claimLease <= 0 {
		return nil, fmt.Errorf("claim lease must be positive")
	}
	return &SendNotificationUseCase{
		repo:           r,
		sender:         s,
		maxSendAttempt: maxSendAttempt,
		claimLease:     claimLease,
	}, nil
}

func (uc *SendNotificationUseCase) Execute(ctx context.Context, input SendNotificationInput) error {
	notification, err := uc.load(ctx, input.NotificationID)
	if err != nil {
		return err
	}
	if notification.Status == domain.StatusSent || notification.Status == domain.StatusFailed {
		return nil
	}
	if notification.DeliveryVersion != input.DeliveryVersion {
		return nil
	}
	if notification.SendAttempts >= uc.maxSendAttempt {
		if err := notification.SetStatus(domain.StatusFailed); err != nil {
			return err
		}
		if err := uc.repo.UpdateNotification(ctx, notification); err != nil {
			return fmt.Errorf("update notification: %w", err)
		}
		return nil
	}

	until := time.Now().UTC().Add(uc.claimLease)
	if err := uc.repo.ClaimNotification(ctx, notification.ID, notification.DeliveryVersion, until); err != nil {
		if errors.Is(err, ErrDeliveryInProgress) || errors.Is(err, ErrNotificationNotFound) {
			return err
		}
		return fmt.Errorf("claim notification: %w", err)
	}
	notification.ClaimedUntil = until

	if err := uc.sender.Send(ctx, notification.Channel, notification.Receiver, notification.Payload); err != nil {
		if incErr := notification.IncreaseSendAttempts(); incErr != nil {
			return incErr
		}
		notification.ReleaseClaim()
		if notification.SendAttempts >= uc.maxSendAttempt {
			if statusErr := notification.SetStatus(domain.StatusFailed); statusErr != nil {
				return statusErr
			}
		}
		if updateErr := uc.repo.UpdateNotification(ctx, notification); updateErr != nil {
			return fmt.Errorf("update notification: %w", updateErr)
		}
		if notification.Status == domain.StatusFailed {
			return nil
		}
		return fmt.Errorf("%w: %w", ErrSendFailed, err)
	}

	if err := notification.IncreaseSendAttempts(); err != nil {
		return err
	}
	if err := notification.SetStatus(domain.StatusSent); err != nil {
		return err
	}
	if err := uc.repo.UpdateNotification(ctx, notification); err != nil {
		return fmt.Errorf("update notification: %w", err)
	}
	return nil
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
