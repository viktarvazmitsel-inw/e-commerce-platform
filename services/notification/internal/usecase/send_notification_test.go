package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/shop/services/notification/internal/domain"
)

type fakeEmailSender struct {
	calls int
	err   error
}

func (f *fakeEmailSender) Send(ctx context.Context, channel, receiver, payload string) error {
	f.calls++
	return f.err
}

func pendingNotification() domain.Notification {
	return domain.Notification{
		ID:       "019a0b2c-7c3e-7a1e-8c3a-9b0e1f2a3b4c",
		Channel:  "email",
		Receiver: "user@example.com",
		Payload:  `{"token":"abc"}`,
		Status:   domain.StatusPending,
	}
}

func TestSendNotificationAttemptLimit(t *testing.T) {
	repo := &fakeNotificationRepository{notifications: []domain.Notification{pendingNotification()}}
	sender := &fakeEmailSender{}
	uc := NewSendNotificationUseCase(repo, sender, 3)

	attempt, err := uc.Execute(context.Background(), SendNotificationInput{
		NotificationID: pendingNotification().ID,
		SendAttempt:    4,
	})
	if attempt != 4 {
		t.Fatalf("attempt: got %d", attempt)
	}
	if !errors.Is(err, ErrSendAttemptLimit) {
		t.Fatalf("error: got %v", err)
	}
	if sender.calls != 0 {
		t.Fatalf("sender calls: got %d", sender.calls)
	}
	stored := repo.notifications[0]
	if stored.Status != domain.StatusFailed || stored.SendAttempts != 4 {
		t.Fatalf("stored: status=%q attempts=%d", stored.Status, stored.SendAttempts)
	}
}

func TestSendNotificationAttemptLimitUpdateError(t *testing.T) {
	repoErr := errors.New("mongo down")
	repo := &fakeNotificationRepository{
		notifications: []domain.Notification{pendingNotification()},
		updateErr:     repoErr,
	}
	sender := &fakeEmailSender{}
	uc := NewSendNotificationUseCase(repo, sender, 3)

	attempt, err := uc.Execute(context.Background(), SendNotificationInput{
		NotificationID: pendingNotification().ID,
		SendAttempt:    4,
	})
	if attempt != 4 {
		t.Fatalf("attempt: got %d", attempt)
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("error: got %v", err)
	}
	if errors.Is(err, ErrSendAttemptLimit) {
		t.Fatal("database error was reported as the attempt limit")
	}
	if sender.calls != 0 {
		t.Fatalf("sender calls: got %d", sender.calls)
	}
	if repo.notifications[0].Status != domain.StatusPending {
		t.Fatalf("status: got %q", repo.notifications[0].Status)
	}
}

func TestSendNotificationNotFound(t *testing.T) {
	sender := &fakeEmailSender{}
	uc := NewSendNotificationUseCase(&fakeNotificationRepository{}, sender, 3)

	attempt, err := uc.Execute(context.Background(), SendNotificationInput{
		NotificationID: "missing",
		SendAttempt:    1,
	})
	if attempt != 1 {
		t.Fatalf("attempt: got %d", attempt)
	}
	if !errors.Is(err, ErrNotificationNotFound) {
		t.Fatalf("error: got %v", err)
	}
	if sender.calls != 0 {
		t.Fatalf("sender calls: got %d", sender.calls)
	}
}

func TestSendNotificationLoadErrorIsRetryable(t *testing.T) {
	repoErr := errors.New("mongo down")
	sender := &fakeEmailSender{}
	uc := NewSendNotificationUseCase(&fakeNotificationRepository{err: repoErr}, sender, 3)

	attempt, err := uc.Execute(context.Background(), SendNotificationInput{
		NotificationID: pendingNotification().ID,
		SendAttempt:    1,
	})
	if attempt != 1 {
		t.Fatalf("attempt: got %d", attempt)
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("error: got %v", err)
	}
	if errors.Is(err, ErrNotificationNotFound) {
		t.Fatal("infrastructure error was reported as not found")
	}
	if sender.calls != 0 {
		t.Fatalf("sender calls: got %d", sender.calls)
	}
}

func TestSendNotificationMarksSent(t *testing.T) {
	repo := &fakeNotificationRepository{notifications: []domain.Notification{pendingNotification()}}
	sender := &fakeEmailSender{}
	uc := NewSendNotificationUseCase(repo, sender, 3)

	attempt, err := uc.Execute(context.Background(), SendNotificationInput{
		NotificationID: pendingNotification().ID,
		SendAttempt:    2,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if attempt != 2 {
		t.Fatalf("attempt: got %d", attempt)
	}
	if sender.calls != 1 {
		t.Fatalf("sender calls: got %d", sender.calls)
	}
	stored := repo.notifications[0]
	if stored.Status != domain.StatusSent || stored.SendAttempts != 2 {
		t.Fatalf("stored: status=%q attempts=%d", stored.Status, stored.SendAttempts)
	}
}

func TestSendNotificationSendFailureReturnsNextAttempt(t *testing.T) {
	sendErr := errors.New("smtp down")
	repoErr := errors.New("mongo down")
	repo := &fakeNotificationRepository{
		notifications: []domain.Notification{pendingNotification()},
		updateErr:     repoErr,
	}
	sender := &fakeEmailSender{err: sendErr}
	uc := NewSendNotificationUseCase(repo, sender, 3)

	attempt, err := uc.Execute(context.Background(), SendNotificationInput{
		NotificationID: pendingNotification().ID,
		SendAttempt:    1,
	})
	if attempt != 2 {
		t.Fatalf("attempt: got %d", attempt)
	}
	if !errors.Is(err, ErrSendFailed) || !errors.Is(err, sendErr) {
		t.Fatalf("error: got %v", err)
	}
	if errors.Is(err, repoErr) {
		t.Fatal("database error replaced the send failure")
	}
	if sender.calls != 1 {
		t.Fatalf("sender calls: got %d", sender.calls)
	}
	if repo.updated == nil || repo.updated.SendAttempts != 2 {
		t.Fatalf("update payload: %+v", repo.updated)
	}
	if repo.notifications[0].Status != domain.StatusPending || repo.notifications[0].SendAttempts != 0 {
		t.Fatalf("stored: status=%q attempts=%d", repo.notifications[0].Status, repo.notifications[0].SendAttempts)
	}
}

func TestSendNotificationSkipsTerminalStatus(t *testing.T) {
	tests := []struct {
		name   string
		status domain.Status
	}{
		{name: "sent", status: domain.StatusSent},
		{name: "failed", status: domain.StatusFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notification := pendingNotification()
			notification.Status = tt.status
			notification.SendAttempts = 2
			repo := &fakeNotificationRepository{notifications: []domain.Notification{notification}}
			sender := &fakeEmailSender{}
			uc := NewSendNotificationUseCase(repo, sender, 3)

			attempt, err := uc.Execute(context.Background(), SendNotificationInput{
				NotificationID: notification.ID,
				SendAttempt:    2,
			})
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if attempt != 2 {
				t.Fatalf("attempt: got %d", attempt)
			}
			if sender.calls != 0 {
				t.Fatalf("sender calls: got %d", sender.calls)
			}
			if repo.updated != nil {
				t.Fatal("terminal notification was updated")
			}
		})
	}
}
