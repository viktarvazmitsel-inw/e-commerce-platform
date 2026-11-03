package broker

import (
	"errors"
	"fmt"
	"testing"

	"github.com/shop/services/notification/internal/domain"
	"github.com/shop/services/notification/internal/usecase"
)

func TestDecodeAuthEvent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		key    string
		body   string
		want   usecase.CreateNotificationInput
		poison bool
	}{
		{
			name: "user created",
			key:  authCreatedKey,
			body: `{"type":"user.created","email":"user@example.com","token":"abc"}`,
			want: usecase.CreateNotificationInput{
				Channel:  "email",
				Receiver: "user@example.com",
				Payload:  `{"type":"user.created","token":"abc"}`,
			},
		},
		{
			name: "password reset",
			key:  authPasswordResetKey,
			body: `{"type":"user.password_reset","email":"user@example.com","token":"reset"}`,
			want: usecase.CreateNotificationInput{
				Channel:  "email",
				Receiver: "user@example.com",
				Payload:  `{"type":"user.password_reset","token":"reset"}`,
			},
		},
		{
			name:   "type mismatches routing key",
			key:    authCreatedKey,
			body:   `{"type":"user.password_reset","email":"user@example.com","token":"abc"}`,
			poison: true,
		},
		{
			name:   "missing token",
			key:    authCreatedKey,
			body:   `{"type":"user.created","email":"user@example.com"}`,
			poison: true,
		},
		{
			name:   "blank email",
			key:    authCreatedKey,
			body:   `{"type":"user.created","email":" ","token":"abc"}`,
			poison: true,
		},
		{
			name:   "invalid json",
			key:    authCreatedKey,
			body:   `{`,
			poison: true,
		},
		{
			name:   "unknown type",
			key:    "user.other",
			body:   `{"type":"user.other","email":"user@example.com","token":"abc"}`,
			poison: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := decodeAuthEvent(tt.key, []byte(tt.body))
			if tt.poison {
				if !errors.Is(err, ErrPoison) {
					t.Fatalf("error: got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got != tt.want {
				t.Fatalf("input: got %+v want %+v", got, tt.want)
			}
		})
	}
}

func TestSettleAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
		err  error
		want disposition
	}{
		{name: "created", id: "id", want: ack},
		{name: "poison", err: ErrPoison, want: deadLetter},
		{name: "invalid receiver", err: domain.ErrInvalidReceiver, want: deadLetter},
		{name: "wrapped validation", err: fmt.Errorf("create: %w", domain.ErrInvalidPayload), want: deadLetter},
		{name: "insert failed", err: errors.New("mongo down"), want: requeue},
		{name: "publish failed after insert", id: "id", err: errors.New("publish"), want: ack},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := settleAuth(tt.id, tt.err); got != tt.want {
				t.Fatalf("disposition: got %d want %d", got, tt.want)
			}
		})
	}
}

func TestDeliveryMessageRoundTrip(t *testing.T) {
	t.Parallel()

	body, err := encodeDelivery("notif-1", 2)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	got, err := decodeDelivery(body)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.NotificationID != "notif-1" || got.DeliveryVersion != 2 {
		t.Fatalf("message: %+v", got)
	}
}

func TestDecodeDeliveryPoison(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: `{`},
		{name: "missing id", body: `{"delivery_version":1}`},
		{name: "version zero", body: `{"notification_id":"id","delivery_version":0}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := decodeDelivery([]byte(tt.body))
			if !errors.Is(err, ErrPoison) {
				t.Fatalf("error: got %v", err)
			}
		})
	}
}

func TestEncodeDeliveryRejectsInvalidMessage(t *testing.T) {
	t.Parallel()

	if _, err := encodeDelivery(" ", 1); err == nil {
		t.Fatal("expected blank id to be rejected")
	}
	if _, err := encodeDelivery("id", 0); err == nil {
		t.Fatal("expected version 0 to be rejected")
	}
}

func TestSettleDelivery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want disposition
	}{
		{name: "sent", want: ack},
		{name: "in progress", err: usecase.ErrDeliveryInProgress, want: ack},
		{name: "missing", err: usecase.ErrNotificationNotFound, want: deadLetter},
		{name: "poison", err: ErrPoison, want: deadLetter},
		{name: "send failed", err: fmt.Errorf("%w: smtp", usecase.ErrSendFailed), want: requeue},
		{name: "database", err: errors.New("mongo down"), want: requeue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := settleDelivery(tt.err); got != tt.want {
				t.Fatalf("disposition: got %d want %d", got, tt.want)
			}
		})
	}
}

func TestNewRejectsEmptyURL(t *testing.T) {
	t.Parallel()

	if _, err := New(" "); err == nil {
		t.Fatal("expected empty url to be rejected")
	}
}
