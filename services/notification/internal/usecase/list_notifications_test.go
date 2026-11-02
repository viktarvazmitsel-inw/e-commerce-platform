package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/shop/services/notification/internal/domain"
)

type fakeNotificationRepository struct {
	notifications []domain.Notification
	total         int
	err           error
	page          int
	quantity      int
	id            string
}

func (f *fakeNotificationRepository) GetNotifications(ctx context.Context, page, quantity int) ([]domain.Notification, int, error) {
	f.page = page
	f.quantity = quantity
	if f.err != nil {
		return nil, 0, f.err
	}
	return f.notifications, f.total, nil
}

func (f *fakeNotificationRepository) GetNotificationByID(ctx context.Context, id string) (*domain.Notification, error) {
	f.id = id
	if f.err != nil {
		return nil, f.err
	}
	for i := range f.notifications {
		if f.notifications[i].ID == id {
			return &f.notifications[i], nil
		}
	}
	return nil, ErrNotificationNotFound
}

func TestListNotificationsExecute(t *testing.T) {
	repo := &fakeNotificationRepository{
		notifications: []domain.Notification{
			{
				ID:           "019a0b2c-7c3e-7a1e-8c3a-9b0e1f2a3b4c",
				Channel:      "email",
				Receiver:     "user@example.com",
				Payload:      `{"token":"secret"}`,
				Status:       domain.StatusSent,
				SendAttempts: 2,
			},
		},
		total: 5,
	}
	uc := NewListNotificationsUseCase(repo)

	out, err := uc.Execute(context.Background(), ListNotificationsInput{Page: 2, Quantity: 10})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if repo.page != 2 || repo.quantity != 10 {
		t.Fatalf("repository called with page=%d quantity=%d", repo.page, repo.quantity)
	}
	if out.Page != 2 || out.Quantity != 10 || out.TotalCount != 5 {
		t.Fatalf("output page=%d quantity=%d total=%d", out.Page, out.Quantity, out.TotalCount)
	}
	if len(out.Notifications) != 1 {
		t.Fatalf("items: got %d", len(out.Notifications))
	}
	item := out.Notifications[0]
	if item.ID != repo.notifications[0].ID ||
		item.Channel != "email" ||
		item.Receiver != "user@example.com" ||
		item.Status != domain.StatusSent ||
		item.SendAttempts != 2 {
		t.Fatalf("item: %+v", item)
	}
}

func TestListNotificationsDefaultsPageAndQuantity(t *testing.T) {
	repo := &fakeNotificationRepository{}
	uc := NewListNotificationsUseCase(repo)

	out, err := uc.Execute(context.Background(), ListNotificationsInput{Page: 0, Quantity: 0})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if repo.page != 1 || repo.quantity != 20 {
		t.Fatalf("repository called with page=%d quantity=%d", repo.page, repo.quantity)
	}
	if out.Page != 1 || out.Quantity != 20 {
		t.Fatalf("output page=%d quantity=%d", out.Page, out.Quantity)
	}
	if out.Notifications == nil {
		t.Fatal("expected an empty slice")
	}

	if _, err := uc.Execute(context.Background(), ListNotificationsInput{Page: -1, Quantity: 101}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if repo.page != 1 || repo.quantity != 20 {
		t.Fatalf("repository called with page=%d quantity=%d", repo.page, repo.quantity)
	}
}

func TestListNotificationsWrapsRepositoryError(t *testing.T) {
	repoErr := errors.New("mongo down")
	uc := NewListNotificationsUseCase(&fakeNotificationRepository{err: repoErr})

	_, err := uc.Execute(context.Background(), ListNotificationsInput{Page: 1, Quantity: 20})
	if !errors.Is(err, repoErr) {
		t.Fatalf("error: got %v", err)
	}
}
