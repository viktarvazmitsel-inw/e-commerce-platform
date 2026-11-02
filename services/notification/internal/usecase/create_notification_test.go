package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/shop/services/notification/internal/domain"
)

type fakeMessageQueuePublisher struct {
	ids []string
	err error
}

func (f *fakeMessageQueuePublisher) PublishForDelivery(ctx context.Context, notificationID string) error {
	f.ids = append(f.ids, notificationID)
	return f.err
}

func TestCreateNotificationExecute(t *testing.T) {
	repo := &fakeNotificationRepository{}
	publisher := &fakeMessageQueuePublisher{}
	uc := NewCreateNotificationUseCase(repo, publisher)

	id, err := uc.Execute(context.Background(), CreateNotificationInput{
		Channel:  " email ",
		Receiver: " user@example.com ",
		Payload:  ` {"token":"abc"} `,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if id == "" {
		t.Fatal("expected notification id")
	}
	if len(repo.notifications) != 1 {
		t.Fatalf("stored: got %d", len(repo.notifications))
	}
	stored := repo.notifications[0]
	if stored.ID != id ||
		stored.Channel != "email" ||
		stored.Receiver != "user@example.com" ||
		stored.Payload != `{"token":"abc"}` ||
		stored.Status != domain.StatusPending ||
		stored.SendAttempts != 0 {
		t.Fatalf("stored: %+v", stored)
	}
	if len(publisher.ids) != 1 || publisher.ids[0] != id {
		t.Fatalf("published ids: %v", publisher.ids)
	}
}

func TestCreateNotificationRepositoryErrorSkipsPublish(t *testing.T) {
	repoErr := errors.New("mongo down")
	repo := &fakeNotificationRepository{createErr: repoErr}
	publisher := &fakeMessageQueuePublisher{}
	uc := NewCreateNotificationUseCase(repo, publisher)

	id, err := uc.Execute(context.Background(), CreateNotificationInput{
		Channel:  "email",
		Receiver: "user@example.com",
		Payload:  `{"token":"abc"}`,
	})
	if id != "" {
		t.Fatalf("id: got %q", id)
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("error: got %v", err)
	}
	if len(repo.notifications) != 0 {
		t.Fatalf("stored: got %d", len(repo.notifications))
	}
	if len(publisher.ids) != 0 {
		t.Fatalf("published ids: %v", publisher.ids)
	}
}

func TestCreateNotificationPublishErrorReturnsID(t *testing.T) {
	publishErr := errors.New("broker down")
	repo := &fakeNotificationRepository{}
	publisher := &fakeMessageQueuePublisher{err: publishErr}
	uc := NewCreateNotificationUseCase(repo, publisher)

	id, err := uc.Execute(context.Background(), CreateNotificationInput{
		Channel:  "email",
		Receiver: "user@example.com",
		Payload:  `{"token":"abc"}`,
	})
	if id == "" {
		t.Fatal("expected notification id")
	}
	if !errors.Is(err, publishErr) {
		t.Fatalf("error: got %v", err)
	}
	if len(repo.notifications) != 1 || repo.notifications[0].ID != id {
		t.Fatalf("stored: %+v", repo.notifications)
	}
	if repo.notifications[0].Status != domain.StatusPending {
		t.Fatalf("status: got %q", repo.notifications[0].Status)
	}
}

func TestCreateNotificationRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateNotificationInput
		wantErr error
	}{
		{
			name:    "channel",
			input:   CreateNotificationInput{Channel: "Email", Receiver: "user@example.com", Payload: `{"token":"abc"}`},
			wantErr: domain.ErrInvalidChannel,
		},
		{
			name:    "receiver",
			input:   CreateNotificationInput{Channel: "email", Receiver: "user", Payload: `{"token":"abc"}`},
			wantErr: domain.ErrInvalidReceiver,
		},
		{
			name:    "payload",
			input:   CreateNotificationInput{Channel: "email", Receiver: "user@example.com", Payload: "null"},
			wantErr: domain.ErrInvalidPayload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeNotificationRepository{}
			publisher := &fakeMessageQueuePublisher{}
			uc := NewCreateNotificationUseCase(repo, publisher)

			id, err := uc.Execute(context.Background(), tt.input)
			if id != "" {
				t.Fatalf("id: got %q", id)
			}
			if err != tt.wantErr {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
			if len(repo.notifications) != 0 {
				t.Fatalf("stored: got %d", len(repo.notifications))
			}
			if len(publisher.ids) != 0 {
				t.Fatalf("published ids: %v", publisher.ids)
			}
		})
	}
}
