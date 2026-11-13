package mail

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSendRendersEventTemplate(t *testing.T) {
	sender := testSender(t)

	var got []byte
	sender.send = func(ctx context.Context, addr, host, from string, to []string, msg []byte) error {
		if addr != "note-mailhog:1025" || host != "note-mailhog" || from != "shop@localhost" {
			t.Fatalf("smtp args addr=%s host=%s from=%s", addr, host, from)
		}
		if len(to) != 1 || to[0] != "user@example.com" {
			t.Fatalf("recipients = %#v", to)
		}
		got = append([]byte(nil), msg...)
		return nil
	}

	err := sender.Send(context.Background(), "auth.user.created", "user@example.com", map[string]any{
		"verification_token": "abc123",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	message := string(got)
	if !strings.Contains(message, "Subject: Confirm your email\r\n") {
		t.Fatalf("message missing subject:\n%s", message)
	}
	if !strings.Contains(message, "abc123") {
		t.Fatalf("message missing token:\n%s", message)
	}
	if !strings.Contains(message, "Content-Type: text/html; charset=UTF-8\r\n") {
		t.Fatalf("message missing html content type:\n%s", message)
	}
}

func TestSendPasswordResetTemplate(t *testing.T) {
	sender := testSender(t)
	var got string
	sender.send = func(ctx context.Context, addr, host, from string, to []string, msg []byte) error {
		got = string(msg)
		return nil
	}

	err := sender.Send(context.Background(), "auth.user.password.reset", "user@example.com", map[string]any{
		"password_reset_token": "reset-1",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !strings.Contains(got, "Subject: Reset your password\r\n") || !strings.Contains(got, "reset-1") {
		t.Fatalf("message = %s", got)
	}
}

func TestSendUnknownEvent(t *testing.T) {
	sender := testSender(t)
	sender.send = func(ctx context.Context, addr, host, from string, to []string, msg []byte) error {
		t.Fatal("send called for unknown event")
		return nil
	}

	err := sender.Send(context.Background(), "auth.user.unknown", "user@example.com", map[string]any{
		"verification_token": "abc123",
	})
	if !errors.Is(err, ErrUnableToSendEmail) {
		t.Fatalf("Send() error = %v, want ErrUnableToSendEmail", err)
	}
}

func TestSendMissingToken(t *testing.T) {
	sender := testSender(t)
	err := sender.Send(context.Background(), "auth.user.created", "user@example.com", map[string]any{})
	if !errors.Is(err, ErrUnableToSendEmail) {
		t.Fatalf("Send() error = %v, want ErrUnableToSendEmail", err)
	}
}

func TestSendWrapsSMTPError(t *testing.T) {
	sender := testSender(t)
	sender.send = func(ctx context.Context, addr, host, from string, to []string, msg []byte) error {
		return errors.New("dial failed")
	}

	err := sender.Send(context.Background(), "auth.user.created", "user@example.com", map[string]any{
		"verification_token": "abc123",
	})
	if !errors.Is(err, ErrUnableToSendEmail) {
		t.Fatalf("Send() error = %v, want ErrUnableToSendEmail", err)
	}
}

func TestNewEmailSenderRejectsEmptyFrom(t *testing.T) {
	if _, err := NewEmailSender("note-mailhog", "1025", ""); err == nil {
		t.Fatal("NewEmailSender() error = nil, want invalid from")
	}
}

func testSender(t *testing.T) *EmailSender {
	t.Helper()
	sender, err := NewEmailSender("note-mailhog", "1025", "shop@localhost")
	if err != nil {
		t.Fatalf("NewEmailSender() error = %v", err)
	}
	return sender
}
