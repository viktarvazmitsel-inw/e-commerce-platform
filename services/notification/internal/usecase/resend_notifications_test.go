package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shop/services/notification/internal/domain"
)

func TestResendNotificationsSupersedesStuckPendingOnce(t *testing.T) {
	window := time.Minute
	repo := &fakeNotificationRepository{
		notifications: []domain.Notification{
			{
				ID:              "pending-1",
				Channel:         "email",
				Receiver:        "user@example.com",
				Status:          domain.StatusPending,
				SendAttempts:    2,
				DeliveryVersion: 2,
				UpdatedAt:       time.Now().UTC().Add(-2 * window),
			},
		},
	}
	publisher := &fakeMessageQueuePublisher{}
	uc := NewResendNotificationsUseCase(repo, publisher, window)

	queued, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if queued != 1 {
		t.Fatalf("queued: got %d", queued)
	}
	stored := repo.notifications[0]
	if stored.Status != domain.StatusPending || stored.SendAttempts != 2 || stored.DeliveryVersion != 3 || stored.AwaitingPublish {
		t.Fatalf("stored: %+v", stored)
	}
	if len(publisher.ids) != 1 || publisher.ids[0] != "pending-1" || publisher.versions[0] != 3 {
		t.Fatalf("published: ids %v versions %v", publisher.ids, publisher.versions)
	}

	queued, err = uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("second Execute: %v", err)
	}
	if queued != 0 {
		t.Fatalf("second queued: got %d", queued)
	}
	if len(publisher.ids) != 1 {
		t.Fatalf("published again: %v", publisher.ids)
	}
	if repo.notifications[0].DeliveryVersion != 3 {
		t.Fatalf("version changed: %+v", repo.notifications[0])
	}
}

func TestResendNotificationsResetsFailedThenPublishes(t *testing.T) {
	repo := &fakeNotificationRepository{
		notifications: []domain.Notification{
			{
				ID:              "failed-1",
				Channel:         "email",
				Receiver:        "user@example.com",
				Status:          domain.StatusFailed,
				SendAttempts:    4,
				DeliveryVersion: 3,
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
	if stored.Status != domain.StatusPending || stored.SendAttempts != 0 || stored.DeliveryVersion != 4 || stored.AwaitingPublish {
		t.Fatalf("stored: %+v", stored)
	}
	if len(publisher.ids) != 1 || publisher.ids[0] != "failed-1" || publisher.versions[0] != 4 {
		t.Fatalf("published: ids %v versions %v", publisher.ids, publisher.versions)
	}
}

func TestResendNotificationsSkipsSent(t *testing.T) {
	repo := &fakeNotificationRepository{
		notifications: []domain.Notification{
			{
				ID:              "sent-1",
				Status:          domain.StatusSent,
				SendAttempts:    1,
				DeliveryVersion: 2,
			},
			{
				ID:              "pending-1",
				Status:          domain.StatusPending,
				SendAttempts:    1,
				DeliveryVersion: 4,
				UpdatedAt:       time.Now().UTC().Add(-2 * time.Minute),
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
	if repo.notifications[0].Status != domain.StatusSent || repo.notifications[0].SendAttempts != 1 || repo.notifications[0].DeliveryVersion != 2 {
		t.Fatalf("sent changed: %+v", repo.notifications[0])
	}
	if len(publisher.ids) != 1 || publisher.ids[0] != "pending-1" || publisher.versions[0] != 5 {
		t.Fatalf("published: ids %v versions %v", publisher.ids, publisher.versions)
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
			{ID: "failed-1", Status: domain.StatusFailed, SendAttempts: 3, DeliveryVersion: 2},
			{ID: "pending-1", Status: domain.StatusPending, SendAttempts: 1, DeliveryVersion: 1, UpdatedAt: time.Now().UTC().Add(-time.Hour)},
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
	if repo.notifications[0].Status != domain.StatusFailed || repo.notifications[0].SendAttempts != 3 || repo.notifications[0].DeliveryVersion != 2 {
		t.Fatalf("failed row changed: %+v", repo.notifications[0])
	}
	if repo.notifications[1].Status != domain.StatusPending || repo.notifications[1].DeliveryVersion != 1 {
		t.Fatalf("later row changed: %+v", repo.notifications[1])
	}
	if len(publisher.ids) != 0 {
		t.Fatalf("published ids: %v", publisher.ids)
	}
}

func TestResendNotificationsPublishErrorStops(t *testing.T) {
	publishErr := errors.New("broker down")
	window := time.Minute
	old := time.Now().UTC().Add(-2 * window)
	repo := &fakeNotificationRepository{
		notifications: []domain.Notification{
			{ID: "pending-1", Status: domain.StatusPending, SendAttempts: 2, DeliveryVersion: 2, UpdatedAt: old},
			{ID: "failed-1", Status: domain.StatusFailed, SendAttempts: 4, DeliveryVersion: 4},
			{ID: "pending-2", Status: domain.StatusPending, SendAttempts: 1, DeliveryVersion: 1, UpdatedAt: old},
		},
	}
	publisher := &fakeMessageQueuePublisher{err: publishErr, failAfter: 1}
	uc := NewResendNotificationsUseCase(repo, publisher, window)

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
	if publisher.versions[0] != 3 || publisher.versions[1] != 5 {
		t.Fatalf("published versions: %v", publisher.versions)
	}
	first := repo.notifications[0]
	if first.Status != domain.StatusPending || first.SendAttempts != 2 || first.DeliveryVersion != 3 || first.AwaitingPublish {
		t.Fatalf("first pending: %+v", first)
	}
	failed := repo.notifications[1]
	if failed.Status != domain.StatusPending || failed.SendAttempts != 0 || failed.DeliveryVersion != 5 || !failed.AwaitingPublish {
		t.Fatalf("failed row after publish error: %+v", failed)
	}
	if repo.notifications[2].Status != domain.StatusPending || repo.notifications[2].DeliveryVersion != 1 {
		t.Fatalf("later pending changed: %+v", repo.notifications[2])
	}
}

func TestResendNotificationsPublishErrorRepublishesSameVersion(t *testing.T) {
	publishErr := errors.New("broker down")
	repo := &fakeNotificationRepository{
		notifications: []domain.Notification{
			{ID: "failed-1", Status: domain.StatusFailed, SendAttempts: 4, DeliveryVersion: 4},
		},
	}
	publisher := &fakeMessageQueuePublisher{err: publishErr}
	uc := NewResendNotificationsUseCase(repo, publisher, time.Minute)

	queued, err := uc.Execute(context.Background())
	if queued != 0 {
		t.Fatalf("queued: got %d", queued)
	}
	if !errors.Is(err, publishErr) {
		t.Fatalf("error: got %v", err)
	}
	stored := repo.notifications[0]
	if stored.Status != domain.StatusPending || stored.SendAttempts != 0 || stored.DeliveryVersion != 5 || !stored.AwaitingPublish {
		t.Fatalf("after failed publish: %+v", stored)
	}

	publisher.err = nil
	queued, err = uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("second Execute: %v", err)
	}
	if queued != 1 {
		t.Fatalf("second queued: got %d", queued)
	}
	if len(publisher.versions) != 2 || publisher.versions[0] != 5 || publisher.versions[1] != 5 {
		t.Fatalf("published versions: %v", publisher.versions)
	}
	stored = repo.notifications[0]
	if stored.DeliveryVersion != 5 || stored.AwaitingPublish || stored.Status != domain.StatusPending {
		t.Fatalf("after republish: %+v", stored)
	}
}

func TestResendNotificationsSkipsFreshAndClaimedPending(t *testing.T) {
	window := time.Minute
	now := time.Now().UTC()
	repo := &fakeNotificationRepository{
		notifications: []domain.Notification{
			{ID: "fresh-pending", Status: domain.StatusPending, UpdatedAt: now, SendAttempts: 1, DeliveryVersion: 1},
			{ID: "old-pending", Status: domain.StatusPending, UpdatedAt: now.Add(-2 * window), SendAttempts: 2, DeliveryVersion: 2},
			{ID: "claimed-pending", Status: domain.StatusPending, UpdatedAt: now.Add(-2 * window), SendAttempts: 1, DeliveryVersion: 3, ClaimedUntil: now.Add(window)},
			{ID: "fresh-failed", Status: domain.StatusFailed, UpdatedAt: now, SendAttempts: 3, DeliveryVersion: 6},
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
	if repo.notifications[0].Status != domain.StatusPending || repo.notifications[0].SendAttempts != 1 || repo.notifications[0].DeliveryVersion != 1 {
		t.Fatalf("fresh pending changed: %+v", repo.notifications[0])
	}
	if repo.notifications[2].DeliveryVersion != 3 || repo.notifications[2].SendAttempts != 1 {
		t.Fatalf("claimed pending changed: %+v", repo.notifications[2])
	}
	if len(publisher.ids) != 2 || publisher.ids[0] != "old-pending" || publisher.ids[1] != "fresh-failed" {
		t.Fatalf("published ids: %v", publisher.ids)
	}
	if publisher.versions[0] != 3 || publisher.versions[1] != 7 {
		t.Fatalf("published versions: %v", publisher.versions)
	}
	if repo.notifications[1].DeliveryVersion != 3 || repo.notifications[1].SendAttempts != 2 || repo.notifications[1].AwaitingPublish {
		t.Fatalf("old pending: %+v", repo.notifications[1])
	}
	if repo.notifications[3].Status != domain.StatusPending || repo.notifications[3].SendAttempts != 0 || repo.notifications[3].DeliveryVersion != 7 {
		t.Fatalf("fresh failed: %+v", repo.notifications[3])
	}
}
