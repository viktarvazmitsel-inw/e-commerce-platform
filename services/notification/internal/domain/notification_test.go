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

func TestIncreaseSendAttempts(t *testing.T) {
	n := mustNotification(t)
	createdAt := n.CreatedAt

	n.IncreaseSendAttempts()
	if n.SendAttempts != 1 {
		t.Fatalf("attempts: got %d", n.SendAttempts)
	}

	n.IncreaseSendAttempts()
	if n.SendAttempts != 2 {
		t.Fatalf("attempts: got %d", n.SendAttempts)
	}
	if n.UpdatedAt.Before(createdAt) {
		t.Fatal("updated timestamp should move forward or stay equal")
	}
	if !n.CreatedAt.Equal(createdAt) {
		t.Fatal("created timestamp should stay unchanged")
	}
}

func TestSetSendAttempts(t *testing.T) {
	n := mustNotification(t)
	createdAt := n.CreatedAt

	if err := n.SetSendAttempts(-1); err != ErrInvalidSendAttempts {
		t.Fatalf("negative attempts: got %v", err)
	}
	if n.SendAttempts != 0 {
		t.Fatalf("attempts changed on rejection: got %d", n.SendAttempts)
	}

	if err := n.SetSendAttempts(4); err != nil {
		t.Fatalf("SetSendAttempts: %v", err)
	}
	if n.SendAttempts != 4 {
		t.Fatalf("attempts: got %d", n.SendAttempts)
	}
	if n.UpdatedAt.Before(createdAt) {
		t.Fatal("updated timestamp should move forward or stay equal")
	}
	if !n.CreatedAt.Equal(createdAt) {
		t.Fatal("created timestamp should stay unchanged")
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
