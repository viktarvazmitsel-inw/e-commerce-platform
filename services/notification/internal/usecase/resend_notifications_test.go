package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shop/services/notification/internal/domain"
)

func TestResendNotificationsPublishesPendingUnchanged(t *testing.T) {
	repo := &fakeNotificationRepository{
		notifications: []domain.Notification{
			{
				ID:           "pending-1",
				Channel:      "email",
				Receiver:     "user@example.com",
				Status:       domain.StatusPending,
				SendAttempts: 2,
			},
		},
	}
	publisher := &fakeMessageQueuePublisher{}
	uc := NewResendNotificationsUseCase(repo, publisher, time.Minute)

	queued, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if queued != 1 {
		t.Fatalf("queued: got %d", queued)
	}
	if repo.updated != nil {
		t.Fatalf("pending notification was updated: %+v", repo.updated)
	}
	stored := repo.notifications[0]
	if stored.Status != domain.StatusPending || stored.SendAttempts != 2 {
		t.Fatalf("stored: %+v", stored)
	}
	if len(publisher.ids) != 1 || publisher.ids[0] != "pending-1" {
		t.Fatalf("published ids: %v", publisher.ids)
	}
}

func TestResendNotificationsResetsFailedThenPublishes(t *testing.T) {
	repo := &fakeNotificationRepository{
		notifications: []domain.Notification{
			{
				ID:           "failed-1",
				Channel:      "email",
				Receiver:     "user@example.com",
				Status:       domain.StatusFailed,
				SendAttempts: 4,
			},
		},
	}
	publisher := &fakeMessageQueuePublisher{}
	uc := NewResendNotificationsUseCase(repo, publisher, time.Minute)

	queued, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if queued != 1 {
		t.Fatalf("queued: got %d", queued)
	}
	stored := repo.notifications[0]
	if stored.Status != domain.StatusPending || stored.SendAttempts != 0 {
		t.Fatalf("stored: %+v", stored)
	}
	if len(publisher.ids) != 1 || publisher.ids[0] != "failed-1" {
		t.Fatalf("published ids: %v", publisher.ids)
	}
}

func TestResendNotificationsSkipsSent(t *testing.T) {
	repo := &fakeNotificationRepository{
		notifications: []domain.Notification{
			{
				ID:           "sent-1",
				Status:       domain.StatusSent,
				SendAttempts: 1,
			},
			{
				ID:           "pending-1",
				Status:       domain.StatusPending,
				SendAttempts: 1,
			},
		},
	}
	publisher := &fakeMessageQueuePublisher{}
	uc := NewResendNotificationsUseCase(repo, publisher, time.Minute)

	queued, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if queued != 1 {
		t.Fatalf("queued: got %d", queued)
	}
	if repo.notifications[0].Status != domain.StatusSent || repo.notifications[0].SendAttempts != 1 {
		t.Fatalf("sent changed: %+v", repo.notifications[0])
	}
	if len(publisher.ids) != 1 || publisher.ids[0] != "pending-1" {
		t.Fatalf("published ids: %v", publisher.ids)
	}
}

func TestResendNotificationsEmpty(t *testing.T) {
	repo := &fakeNotificationRepository{}
	publisher := &fakeMessageQueuePublisher{}
	uc := NewResendNotificationsUseCase(repo, publisher, time.Minute)

	queued, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if queued != 0 {
		t.Fatalf("queued: got %d", queued)
	}
	if len(publisher.ids) != 0 {
		t.Fatalf("published ids: %v", publisher.ids)
	}
}

func TestResendNotificationsRepositoryErrorSkipsPublish(t *testing.T) {
	repoErr := errors.New("mongo down")
	repo := &fakeNotificationRepository{
		notifications: []domain.Notification{{ID: "pending-1", Status: domain.StatusPending}},
		err:           repoErr,
	}
	publisher := &fakeMessageQueuePublisher{}
	uc := NewResendNotificationsUseCase(repo, publisher, time.Minute)

	queued, err := uc.Execute(context.Background())
	if queued != 0 {
		t.Fatalf("queued: got %d", queued)
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("error: got %v", err)
	}
	if len(publisher.ids) != 0 {
		t.Fatalf("published ids: %v", publisher.ids)
	}
}

func TestResendNotificationsUpdateErrorStops(t *testing.T) {
	updateErr := errors.New("write failed")
	repo := &fakeNotificationRepository{
		notifications: []domain.Notification{
			{ID: "failed-1", Status: domain.StatusFailed, SendAttempts: 3},
			{ID: "pending-1", Status: domain.StatusPending, SendAttempts: 1},
		},
		updateErr: updateErr,
	}
	publisher := &fakeMessageQueuePublisher{}
	uc := NewResendNotificationsUseCase(repo, publisher, time.Minute)

	queued, err := uc.Execute(context.Background())
	if queued != 0 {
		t.Fatalf("queued: got %d", queued)
	}
	if !errors.Is(err, updateErr) {
		t.Fatalf("error: got %v", err)
	}
	if repo.notifications[0].Status != domain.StatusFailed || repo.notifications[0].SendAttempts != 3 {
		t.Fatalf("failed row changed: %+v", repo.notifications[0])
	}
	if repo.notifications[1].Status != domain.StatusPending {
		t.Fatalf("later row changed: %+v", repo.notifications[1])
	}
	if len(publisher.ids) != 0 {
		t.Fatalf("published ids: %v", publisher.ids)
	}
}

func TestResendNotificationsPublishErrorStops(t *testing.T) {
	publishErr := errors.New("broker down")
	repo := &fakeNotificationRepository{
		notifications: []domain.Notification{
			{ID: "pending-1", Status: domain.StatusPending, SendAttempts: 2},
			{ID: "failed-1", Status: domain.StatusFailed, SendAttempts: 4},
			{ID: "pending-2", Status: domain.StatusPending, SendAttempts: 1},
		},
	}
	publisher := &fakeMessageQueuePublisher{err: publishErr, failAfter: 1}
	uc := NewResendNotificationsUseCase(repo, publisher, time.Minute)

	queued, err := uc.Execute(context.Background())
	if queued != 1 {
		t.Fatalf("queued: got %d", queued)
	}
	if !errors.Is(err, publishErr) {
		t.Fatalf("error: got %v", err)
	}
	if len(publisher.ids) != 2 || publisher.ids[0] != "pending-1" || publisher.ids[1] != "failed-1" {
		t.Fatalf("published ids: %v", publisher.ids)
	}
	if repo.notifications[0].Status != domain.StatusPending || repo.notifications[0].SendAttempts != 2 {
		t.Fatalf("first pending changed: %+v", repo.notifications[0])
	}
	if repo.notifications[1].Status != domain.StatusPending || repo.notifications[1].SendAttempts != 0 {
		t.Fatalf("failed row after publish error: %+v", repo.notifications[1])
	}
	if repo.notifications[2].Status != domain.StatusPending || repo.notifications[2].SendAttempts != 1 {
		t.Fatalf("later pending changed: %+v", repo.notifications[2])
	}
}

func TestResendNotificationsSkipsFreshPending(t *testing.T) {
	window := time.Minute
	now := time.Now().UTC()
	repo := &fakeNotificationRepository{
		notifications: []domain.Notification{
			{ID: "fresh-pending", Status: domain.StatusPending, CreatedAt: now, SendAttempts: 1},
			{ID: "old-pending", Status: domain.StatusPending, CreatedAt: now.Add(-2 * window), SendAttempts: 2},
			{ID: "fresh-failed", Status: domain.StatusFailed, CreatedAt: now, SendAttempts: 3},
		},
	}
	publisher := &fakeMessageQueuePublisher{}
	uc := NewResendNotificationsUseCase(repo, publisher, window)

	queued, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if queued != 2 {
		t.Fatalf("queued: got %d", queued)
	}
	if repo.retryWindow != window {
		t.Fatalf("retry window: got %v", repo.retryWindow)
	}
	if repo.notifications[0].Status != domain.StatusPending || repo.notifications[0].SendAttempts != 1 {
		t.Fatalf("fresh pending changed: %+v", repo.notifications[0])
	}
	if len(publisher.ids) != 2 || publisher.ids[0] != "old-pending" || publisher.ids[1] != "fresh-failed" {
		t.Fatalf("published ids: %v", publisher.ids)
	}
	if repo.notifications[2].Status != domain.StatusPending || repo.notifications[2].SendAttempts != 0 {
		t.Fatalf("fresh failed: %+v", repo.notifications[2])
	}
}
