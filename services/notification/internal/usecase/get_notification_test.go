package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shop/services/notification/internal/domain"
)

func TestGetNotificationExecute(t *testing.T) {
	createdAt := time.Date(2026, 11, 2, 8, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 11, 2, 9, 30, 0, 0, time.UTC)
	repo := &fakeNotificationRepository{
		notifications: []domain.Notification{
			{
				ID:           "019a0b2c-7c3e-7a1e-8c3a-9b0e1f2a3b4c",
				Channel:      "email",
				Receiver:     "user@example.com",
				Payload:      `{"subject":"hello"}`,
				Status:       domain.StatusSent,
				SendAttempts: 2,
				CreatedAt:    createdAt,
				UpdatedAt:    updatedAt,
			},
		},
	}
	uc := NewGetNotificationUseCase(repo)

	out, err := uc.Execute(context.Background(), GetNotificationInput{
		NotificationID: "019a0b2c-7c3e-7a1e-8c3a-9b0e1f2a3b4c",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if repo.id != "019a0b2c-7c3e-7a1e-8c3a-9b0e1f2a3b4c" {
		t.Fatalf("repository called with id %q", repo.id)
	}
	if out.ID != repo.notifications[0].ID ||
		out.Channel != "email" ||
		out.Receiver != "user@example.com" ||
		out.Payload != `{"subject":"hello"}` ||
		out.Status != domain.StatusSent ||
		out.SendAttempts != 2 ||
		!out.CreatedAt.Equal(createdAt) ||
		!out.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("output: %+v", out)
	}
}

func TestGetNotificationNotFound(t *testing.T) {
	uc := NewGetNotificationUseCase(&fakeNotificationRepository{})

	_, err := uc.Execute(context.Background(), GetNotificationInput{NotificationID: "missing"})
	if !errors.Is(err, ErrNotificationNotFound) {
		t.Fatalf("error: got %v", err)
	}
}

func TestGetNotificationWrapsRepositoryError(t *testing.T) {
	repoErr := errors.New("mongo down")
	uc := NewGetNotificationUseCase(&fakeNotificationRepository{err: repoErr})

	_, err := uc.Execute(context.Background(), GetNotificationInput{NotificationID: "019a0b2c-7c3e-7a1e-8c3a-9b0e1f2a3b4c"})
	if !errors.Is(err, repoErr) {
		t.Fatalf("error: got %v", err)
	}
	if errors.Is(err, ErrNotificationNotFound) {
		t.Fatal("infrastructure error was reported as not found")
	}
}
