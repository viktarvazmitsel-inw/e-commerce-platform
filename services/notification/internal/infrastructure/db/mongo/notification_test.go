package mongo

import (
	"testing"
	"time"

	"github.com/shop/services/notification/internal/domain"
)

func TestNotificationDocumentRoundTrip(t *testing.T) {
	claimedUntil := time.Date(2026, 11, 3, 12, 0, 0, 0, time.FixedZone("CET", 3600))
	createdAt := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 11, 3, 11, 0, 0, 0, time.UTC)
	original := &domain.Notification{
		ID:              "019a0b2c-7c3e-7a1e-8c3a-9b0e1f2a3b4c",
		Channel:         "email",
		Receiver:        "user@example.com",
		Payload:         `{"token":"abc"}`,
		Status:          domain.StatusPending,
		SendAttempts:    2,
		DeliveryVersion: 3,
		ClaimedUntil:    claimedUntil,
		AwaitingPublish: true,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}

	got := FromDomain(original).ToDomain()
	if got.ID != original.ID || got.Channel != original.Channel || got.Receiver != original.Receiver || got.Payload != original.Payload {
		t.Fatalf("identity fields: %+v", got)
	}
	if got.Status != original.Status || got.SendAttempts != original.SendAttempts || got.DeliveryVersion != original.DeliveryVersion {
		t.Fatalf("delivery fields: %+v", got)
	}
	if !got.AwaitingPublish {
		t.Fatal("awaiting publish was cleared")
	}
	if !got.ClaimedUntil.Equal(claimedUntil.UTC()) || !got.CreatedAt.Equal(createdAt) || !got.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("times: claimed=%s created=%s updated=%s", got.ClaimedUntil, got.CreatedAt, got.UpdatedAt)
	}
}

func TestNotificationDocumentZeroClaim(t *testing.T) {
	original := &domain.Notification{
		ID:              "019a0b2c-7c3e-7a1e-8c3a-9b0e1f2a3b4c",
		Channel:         "email",
		Receiver:        "user@example.com",
		Payload:         `{"token":"abc"}`,
		Status:          domain.StatusFailed,
		DeliveryVersion: 1,
		CreatedAt:       time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 11, 3, 11, 0, 0, 0, time.UTC),
	}

	got := FromDomain(original).ToDomain()
	if !got.ClaimedUntil.IsZero() {
		t.Fatalf("claimed until: %s", got.ClaimedUntil)
	}
	if got.AwaitingPublish {
		t.Fatal("awaiting publish was set")
	}
	if got.Status != domain.StatusFailed {
		t.Fatalf("status: %s", got.Status)
	}
}
