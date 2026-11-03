package domain

import (
	"strings"
	"testing"
	"time"
)

func TestNewNotification(t *testing.T) {
	before := time.Now().UTC()
	n, err := NewNotification(" verification ", " User@Example.com ", ` {"token":"abc"} `)
	if err != nil {
		t.Fatalf("NewNotification: %v", err)
	}

	if n.ID == "" {
		t.Fatal("expected generated id")
	}
	if n.Channel != "verification" {
		t.Fatalf("channel: got %q", n.Channel)
	}
	if n.Receiver != "User@Example.com" {
		t.Fatalf("receiver: got %q", n.Receiver)
	}
	if n.Payload != `{"token":"abc"}` {
		t.Fatalf("payload: got %q", n.Payload)
	}
	if n.Status != StatusPending {
		t.Fatalf("status: got %q", n.Status)
	}
	if n.SendAttempts != 0 {
		t.Fatalf("send attempts: got %d", n.SendAttempts)
	}
	if n.DeliveryVersion != 1 {
		t.Fatalf("delivery version: got %d", n.DeliveryVersion)
	}
	if !n.AwaitingPublish {
		t.Fatal("new notification should await publish")
	}
	if !n.ClaimedUntil.IsZero() {
		t.Fatal("new notification should not be claimed")
	}
	if n.CreatedAt.Before(before) || n.UpdatedAt.Before(before) {
		t.Fatal("timestamps should be set to the current time")
	}
	if !n.CreatedAt.Equal(n.UpdatedAt) {
		t.Fatal("created and updated timestamps should match")
	}
}

func TestNewNotificationValidation(t *testing.T) {
	validPayload := `{"token":"abc"}`
	tests := []struct {
		name     string
		channel  string
		receiver string
		payload  string
		wantErr  error
	}{
		{name: "empty channel", channel: " ", receiver: "user@example.com", payload: validPayload, wantErr: ErrInvalidChannel},
		{name: "channel with spaces", channel: "email queue", receiver: "user@example.com", payload: validPayload, wantErr: ErrInvalidChannel},
		{name: "uppercase channel", channel: "Email", receiver: "user@example.com", payload: validPayload, wantErr: ErrInvalidChannel},
		{name: "empty receiver", channel: "email", receiver: " ", payload: validPayload, wantErr: ErrInvalidReceiver},
		{name: "receiver without domain", channel: "email", receiver: "user", payload: validPayload, wantErr: ErrInvalidReceiver},
		{name: "empty payload", channel: "email", receiver: "user@example.com", payload: " ", wantErr: ErrInvalidPayload},
		{name: "invalid json", channel: "email", receiver: "user@example.com", payload: "{", wantErr: ErrInvalidPayload},
		{name: "json null", channel: "email", receiver: "user@example.com", payload: "null", wantErr: ErrInvalidPayload},
		{name: "json string", channel: "email", receiver: "user@example.com", payload: `"token"`, wantErr: ErrInvalidPayload},
		{name: "payload too large", channel: "email", receiver: "user@example.com", payload: `{"token":"` + strings.Repeat("a", maxPayloadBytes) + `"}`, wantErr: ErrInvalidPayload},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewNotification(tt.channel, tt.receiver, tt.payload)
			if err != tt.wantErr {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestSetStatus(t *testing.T) {
	n := mustNotification(t)

	if err := n.SetStatus(Status("unknown")); err != ErrInvalidStatus {
		t.Fatalf("unknown status: got %v", err)
	}
	if err := n.SetStatus(StatusPending); err != ErrInvalidStatusTransition {
		t.Fatalf("pending to pending: got %v", err)
	}
	if err := n.SetStatus(StatusSent); err != nil {
		t.Fatalf("pending to sent: %v", err)
	}
	if n.Status != StatusSent {
		t.Fatalf("status: got %q", n.Status)
	}
	if err := n.SetStatus(StatusFailed); err != ErrInvalidStatusTransition {
		t.Fatalf("sent to failed: got %v", err)
	}

	failed := mustNotification(t)
	if err := failed.SetStatus(StatusFailed); err != nil {
		t.Fatalf("pending to failed: %v", err)
	}
	if failed.Status != StatusFailed {
		t.Fatalf("status: got %q", failed.Status)
	}
}

func TestResetForResend(t *testing.T) {
	failed := mustNotification(t)
	if err := failed.SetStatus(StatusFailed); err != nil {
		t.Fatalf("pending to failed: %v", err)
	}
	failed.SendAttempts = 3
	failed.DeliveryVersion = 2
	failed.ClaimedUntil = time.Now().UTC().Add(time.Minute)
	failed.AwaitingPublish = false
	createdAt := failed.CreatedAt
	updatedAt := failed.UpdatedAt

	if err := failed.ResetForResend(); err != nil {
		t.Fatalf("ResetForResend: %v", err)
	}
	if failed.Status != StatusPending {
		t.Fatalf("status: got %q", failed.Status)
	}
	if failed.SendAttempts != 0 {
		t.Fatalf("attempts: got %d", failed.SendAttempts)
	}
	if failed.DeliveryVersion != 3 {
		t.Fatalf("delivery version: got %d", failed.DeliveryVersion)
	}
	if !failed.ClaimedUntil.IsZero() {
		t.Fatal("claim should be cleared")
	}
	if !failed.AwaitingPublish {
		t.Fatal("reset should await publish")
	}
	if failed.UpdatedAt.Before(updatedAt) {
		t.Fatal("updated timestamp should move forward or stay equal")
	}
	if !failed.CreatedAt.Equal(createdAt) {
		t.Fatal("created timestamp should stay unchanged")
	}

	pending := mustNotification(t)
	if err := pending.ResetForResend(); err != ErrInvalidStatusTransition {
		t.Fatalf("pending: got %v", err)
	}
	if pending.Status != StatusPending || pending.SendAttempts != 0 {
		t.Fatalf("pending changed: %+v", pending)
	}

	sent := mustNotification(t)
	if err := sent.SetStatus(StatusSent); err != nil {
		t.Fatalf("pending to sent: %v", err)
	}
	if err := sent.ResetForResend(); err != ErrInvalidStatusTransition {
		t.Fatalf("sent: got %v", err)
	}
	if sent.Status != StatusSent {
		t.Fatalf("status: got %q", sent.Status)
	}
}

func TestSupersedeDelivery(t *testing.T) {
	pending := mustNotification(t)
	if err := pending.IncreaseSendAttempts(); err != nil {
		t.Fatalf("IncreaseSendAttempts: %v", err)
	}
	pending.DeliveryVersion = 3
	pending.ClaimedUntil = time.Now().UTC().Add(time.Minute)
	pending.AwaitingPublish = false
	createdAt := pending.CreatedAt

	if err := pending.SupersedeDelivery(); err != nil {
		t.Fatalf("SupersedeDelivery: %v", err)
	}
	if pending.Status != StatusPending {
		t.Fatalf("status: got %q", pending.Status)
	}
	if pending.SendAttempts != 1 {
		t.Fatalf("attempts: got %d", pending.SendAttempts)
	}
	if pending.DeliveryVersion != 4 {
		t.Fatalf("delivery version: got %d", pending.DeliveryVersion)
	}
	if !pending.ClaimedUntil.IsZero() {
		t.Fatal("claim should be cleared")
	}
	if !pending.AwaitingPublish {
		t.Fatal("supersede should await publish")
	}
	if !pending.CreatedAt.Equal(createdAt) {
		t.Fatal("created timestamp should stay unchanged")
	}

	failed := mustNotification(t)
	if err := failed.SetStatus(StatusFailed); err != nil {
		t.Fatalf("pending to failed: %v", err)
	}
	if err := failed.SupersedeDelivery(); err != ErrInvalidStatusTransition {
		t.Fatalf("failed: got %v", err)
	}

	sent := mustNotification(t)
	if err := sent.SetStatus(StatusSent); err != nil {
		t.Fatalf("pending to sent: %v", err)
	}
	if err := sent.SupersedeDelivery(); err != ErrInvalidStatusTransition {
		t.Fatalf("sent: got %v", err)
	}
	if sent.DeliveryVersion != 1 {
		t.Fatalf("delivery version: got %d", sent.DeliveryVersion)
	}
}

func TestIncreaseSendAttempts(t *testing.T) {
	n := mustNotification(t)
	createdAt := n.CreatedAt

	if err := n.IncreaseSendAttempts(); err != nil {
		t.Fatalf("IncreaseSendAttempts: %v", err)
	}
	if n.SendAttempts != 1 {
		t.Fatalf("attempts: got %d", n.SendAttempts)
	}

	if err := n.IncreaseSendAttempts(); err != nil {
		t.Fatalf("IncreaseSendAttempts: %v", err)
	}
	if n.SendAttempts != 2 {
		t.Fatalf("attempts: got %d", n.SendAttempts)
	}
	if n.UpdatedAt.Before(createdAt) {
		t.Fatal("updated timestamp should move forward or stay equal")
	}
	if !n.CreatedAt.Equal(createdAt) {
		t.Fatal("created timestamp should stay unchanged")
	}

	if err := n.SetStatus(StatusSent); err != nil {
		t.Fatalf("pending to sent: %v", err)
	}
	if err := n.IncreaseSendAttempts(); err != ErrInvalidStatusTransition {
		t.Fatalf("sent: got %v", err)
	}
	if n.SendAttempts != 2 {
		t.Fatalf("attempts changed on rejection: got %d", n.SendAttempts)
	}
}

func mustNotification(t *testing.T) *Notification {
	t.Helper()
	n, err := NewNotification("email", "user@example.com", `{"token":"abc"}`)
	if err != nil {
		t.Fatalf("NewNotification: %v", err)
	}
	return n
}
