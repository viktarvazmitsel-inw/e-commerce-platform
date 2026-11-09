package usecase

import (
	"context"
	"fmt"

	"github.com/shop/services/notification/internal/domain"
)

type ListNotificationsInput struct {
	Page     int
	Quantity int
}

type NotificationListItem struct {
	ID             string        `json:"id"`
	Event          string        `json:"event"`
	Receiver       string        `json:"receiver"`
	Status         domain.Status `json:"status"`
	ResendAttempts int           `json:"resend_attempts"`
}

type ListNotificationsOutput struct {
	Notifications []NotificationListItem `json:"notifications"`
	Page          int                    `json:"page"`
	Quantity      int                    `json:"quantity"`
	TotalCount    int                    `json:"total_count"`
}

type ListNotificationsUseCase struct {
	repo NotificationRepository
}

func NewListNotificationsUseCase(r NotificationRepository) *ListNotificationsUseCase {
	return &ListNotificationsUseCase{repo: r}
}

func (uc *ListNotificationsUseCase) Execute(ctx context.Context, input ListNotificationsInput) (*ListNotificationsOutput, error) {
	page := input.Page
	if page <= 0 {
		page = 1
	}

	quantity := input.Quantity
	if quantity <= 0 || quantity > 100 {
		quantity = 20
	}

	notifications, total, err := uc.repo.GetNotifications(ctx, page, quantity)
	if err != nil {
		return nil, fmt.Errorf("load notifications: %w", err)
	}

	items := make([]NotificationListItem, 0, len(notifications))
	for _, n := range notifications {
		items = append(items, NotificationListItem{
			ID:             n.ID,
			Event:          n.Event,
			Receiver:       n.Receiver,
			Status:         n.Status,
			ResendAttempts: n.ResendAttempts,
		})
	}

	return &ListNotificationsOutput{
		Notifications: items,
		Page:          page,
		Quantity:      quantity,
		TotalCount:    total,
	}, nil
}
