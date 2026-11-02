package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shop/services/notification/internal/domain"
)

type GetNotificationInput struct {
	NotificationID string
}

type GetNotificationOutput struct {
	ID           string        `json:"id"`
	Channel      string        `json:"channel"`
	Receiver     string        `json:"receiver"`
	Payload      string        `json:"payload"`
	Status       domain.Status `json:"status"`
	SendAttempts int           `json:"send_attempts"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

type GetNotificationUseCase struct {
	repo NotificationRepository
}

func NewGetNotificationUseCase(r NotificationRepository) *GetNotificationUseCase {
	return &GetNotificationUseCase{repo: r}
}

func (uc *GetNotificationUseCase) Execute(ctx context.Context, input GetNotificationInput) (*GetNotificationOutput, error) {
	notification, err := uc.repo.GetNotificationByID(ctx, input.NotificationID)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			return nil, ErrNotificationNotFound
		}
		return nil, fmt.Errorf("load notification: %w", err)
	}
	if notification == nil {
		return nil, ErrNotificationNotFound
	}

	return &GetNotificationOutput{
		ID:           notification.ID,
		Channel:      notification.Channel,
		Receiver:     notification.Receiver,
		Payload:      notification.Payload,
		Status:       notification.Status,
		SendAttempts: notification.SendAttempts,
		CreatedAt:    notification.CreatedAt,
		UpdatedAt:    notification.UpdatedAt,
	}, nil
}
