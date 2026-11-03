package mongo

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/shop/services/notification/internal/domain"
	"github.com/shop/services/notification/internal/usecase"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func newTestRepository(t *testing.T) *NotificationRepository {
	t.Helper()
	uri := os.Getenv("NOTIFICATION_MONGO_URI")
	if uri == "" {
		t.Skip("NOTIFICATION_MONGO_URI is not set")
	}

	ctx := context.Background()
	client, err := drivermongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Disconnect(context.Background())
	})

	db := client.Database("notification_repository_test")
	if err := db.Collection(notificationsCollection).Drop(ctx); err != nil {
		t.Fatalf("drop collection: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Collection(notificationsCollection).Drop(context.Background())
	})

	repo, err := NewNotificationRepository(db)
	if err != nil {
		t.Fatalf("NewNotificationRepository: %v", err)
	}
	return repo
}

func TestRepositoryInsertAndGet(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	notification := sampleNotification("n-1", domain.StatusPending, time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC))

	if err := repo.CreateNotification(ctx, &notification); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetNotificationByID(ctx, notification.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Receiver != notification.Receiver || got.Payload != notification.Payload || got.Status != domain.StatusPending {
		t.Fatalf("got %+v", got)
	}
	if !got.ClaimedUntil.IsZero() {
		t.Fatalf("claimed until: %s", got.ClaimedUntil)
	}

	notification.Status = domain.StatusSent
	notification.SendAttempts = 1
	notification.AwaitingPublish = false
	notification.UpdatedAt = notification.UpdatedAt.Add(time.Minute)
	if err := repo.UpdateNotification(ctx, &notification); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err = repo.GetNotificationByID(ctx, notification.ID)
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if got.Status != domain.StatusSent || got.SendAttempts != 1 || got.AwaitingPublish {
		t.Fatalf("updated: %+v", got)
	}
}

func TestRepositoryMissingID(t *testing.T) {
	repo := newTestRepository(t)
	_, err := repo.GetNotificationByID(context.Background(), "missing")
	if !errors.Is(err, usecase.ErrNotificationNotFound) {
		t.Fatalf("error: %v", err)
	}
}

func TestRepositoryListOrderAndTotal(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	older := sampleNotification("n-old", domain.StatusSent, time.Date(2026, 11, 3, 9, 0, 0, 0, time.UTC))
	newer := sampleNotification("n-new", domain.StatusPending, time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC))
	if err := repo.CreateNotification(ctx, &older); err != nil {
		t.Fatalf("create older: %v", err)
	}
	if err := repo.CreateNotification(ctx, &newer); err != nil {
		t.Fatalf("create newer: %v", err)
	}

	page, total, err := repo.GetNotifications(ctx, 1, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 2 {
		t.Fatalf("total: %d", total)
	}
	if len(page) != 1 || page[0].ID != newer.ID {
		t.Fatalf("page: %+v", page)
	}
}

func TestRepositoryStatusFilter(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	failed := sampleNotification("failed", domain.StatusFailed, now.Add(-time.Hour))
	awaiting := sampleNotification("awaiting", domain.StatusPending, now.Add(-time.Minute))
	awaiting.AwaitingPublish = true

	claimed := sampleNotification("claimed", domain.StatusPending, now.Add(-time.Hour))
	claimed.AwaitingPublish = true
	claimed.ClaimedUntil = now.Add(time.Minute)

	fresh := sampleNotification("fresh", domain.StatusPending, now.Add(-time.Second))
	fresh.AwaitingPublish = false

	stuck := sampleNotification("stuck", domain.StatusPending, now.Add(-2*time.Hour))
	stuck.AwaitingPublish = false

	sent := sampleNotification("sent", domain.StatusSent, now)
	for _, notification := range []*domain.Notification{&failed, &awaiting, &claimed, &fresh, &stuck, &sent} {
		if err := repo.CreateNotification(ctx, notification); err != nil {
			t.Fatalf("create %s: %v", notification.ID, err)
		}
	}

	got, err := repo.GetNotificationsByStatuses(ctx, []domain.Status{domain.StatusPending, domain.StatusFailed}, time.Hour)
	if err != nil {
		t.Fatalf("filter: %v", err)
	}

	ids := map[string]struct{}{}
	for _, notification := range got {
		ids[notification.ID] = struct{}{}
	}
	for _, id := range []string{failed.ID, awaiting.ID, stuck.ID} {
		if _, ok := ids[id]; !ok {
			t.Fatalf("missing %s in %+v", id, got)
		}
	}
	for _, id := range []string{claimed.ID, fresh.ID, sent.ID} {
		if _, ok := ids[id]; ok {
			t.Fatalf("unexpected %s in %+v", id, got)
		}
	}
}

func TestRepositoryClaim(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	notification := sampleNotification("claim", domain.StatusPending, time.Now().UTC().Truncate(time.Millisecond))
	if err := repo.CreateNotification(ctx, &notification); err != nil {
		t.Fatalf("create: %v", err)
	}

	until := time.Now().UTC().Add(time.Minute).Truncate(time.Millisecond)
	if err := repo.ClaimNotification(ctx, notification.ID, notification.DeliveryVersion, until); err != nil {
		t.Fatalf("claim: %v", err)
	}
	got, err := repo.GetNotificationByID(ctx, notification.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !got.ClaimedUntil.Equal(until) {
		t.Fatalf("claimed until: got %s want %s", got.ClaimedUntil, until)
	}

	err = repo.ClaimNotification(ctx, notification.ID, notification.DeliveryVersion, until.Add(time.Minute))
	if !errors.Is(err, usecase.ErrDeliveryInProgress) {
		t.Fatalf("second claim: %v", err)
	}

	err = repo.ClaimNotification(ctx, "missing", 1, until)
	if !errors.Is(err, usecase.ErrNotificationNotFound) {
		t.Fatalf("missing claim: %v", err)
	}
}

func TestRepositoryMarkDeliveryPublished(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	notification := sampleNotification("publish", domain.StatusPending, time.Now().UTC().Truncate(time.Millisecond))
	notification.AwaitingPublish = true
	notification.DeliveryVersion = 2
	if err := repo.CreateNotification(ctx, &notification); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := repo.MarkDeliveryPublished(ctx, notification.ID, 1); err != nil {
		t.Fatalf("older version: %v", err)
	}
	got, err := repo.GetNotificationByID(ctx, notification.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !got.AwaitingPublish {
		t.Fatal("older version cleared awaiting publish")
	}

	if err := repo.MarkDeliveryPublished(ctx, notification.ID, 2); err != nil {
		t.Fatalf("current version: %v", err)
	}
	got, err = repo.GetNotificationByID(ctx, notification.ID)
	if err != nil {
		t.Fatalf("get after publish: %v", err)
	}
	if got.AwaitingPublish {
		t.Fatal("awaiting publish was not cleared")
	}

	err = repo.MarkDeliveryPublished(ctx, "missing", 2)
	if !errors.Is(err, usecase.ErrNotificationNotFound) {
		t.Fatalf("missing publish: %v", err)
	}
}

func sampleNotification(id string, status domain.Status, createdAt time.Time) domain.Notification {
	return domain.Notification{
		ID:              id,
		Channel:         "email",
		Receiver:        "user@example.com",
		Payload:         `{"token":"abc"}`,
		Status:          status,
		DeliveryVersion: 1,
		AwaitingPublish: status == domain.StatusPending,
		CreatedAt:       createdAt.UTC(),
		UpdatedAt:       createdAt.UTC(),
	}
}
