package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shop/services/notification/internal/domain"
)

type fakeEmailSender struct {
	calls int
	err   error
	hook  func()
}

func (f *fakeEmailSender) Send(ctx context.Context, channel, receiver, payload string) error {
	f.calls++
	if f.hook != nil {
		f.hook()
	}
	return f.err
}

func pendingNotification() domain.Notification {
	return domain.Notification{
		ID:              "019a0b2c-7c3e-7a1e-8c3a-9b0e1f2a3b4c",
		Channel:         "email",
		Receiver:        "user@example.com",
		Payload:         `{"token":"abc"}`,
		Status:          domain.StatusPending,
		DeliveryVersion: 1,
	}
}

func newSendUseCase(t *testing.T, repo NotificationRepository, sender EmailSender, maxSendAttempt int) *SendNotificationUseCase {
	t.Helper()
	uc, err := NewSendNotificationUseCase(repo, sender, maxSendAttempt, time.Minute)
	if err != nil {
		t.Fatalf("NewSendNotificationUseCase: %v", err)
	}
	return uc
}

func TestNewSendNotificationUseCaseRejectsInvalidLimits(t *testing.T) {
	if _, err := NewSendNotificationUseCase(nil, nil, 0, time.Minute); err == nil {
		t.Fatal("expected max send attempt to be rejected")
	}
	if _, err := NewSendNotificationUseCase(nil, nil, 3, 0); err == nil {
		t.Fatal("expected claim lease to be rejected")
	}
}

func TestSendNotificationAttemptLimit(t *testing.T) {
	notification := pendingNotification()
	notification.SendAttempts = 3
	repo := &fakeNotificationRepository{notifications: []domain.Notification{notification}}
	sender := &fakeEmailSender{}
	uc := newSendUseCase(t, repo, sender, 3)

	err := uc.Execute(context.Background(), SendNotificationInput{
		NotificationID:  notification.ID,
		DeliveryVersion: 1,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if sender.calls != 0 {
		t.Fatalf("sender calls: got %d", sender.calls)
	}
	stored := repo.notifications[0]
	if stored.Status != domain.StatusFailed || stored.SendAttempts != 3 {
		t.Fatalf("stored: status=%q attempts=%d", stored.Status, stored.SendAttempts)
	}
}

func TestSendNotificationAttemptLimitUpdateError(t *testing.T) {
	repoErr := errors.New("mongo down")
	notification := pendingNotification()
	notification.SendAttempts = 3
	repo := &fakeNotificationRepository{
		notifications: []domain.Notification{notification},
		updateErr:     repoErr,
	}
	sender := &fakeEmailSender{}
	uc := newSendUseCase(t, repo, sender, 3)

	err := uc.Execute(context.Background(), SendNotificationInput{
		NotificationID:  notification.ID,
		DeliveryVersion: 1,
	})
	if !errors.Is(err, repoErr) {
		t.Fatalf("error: got %v", err)
	}
	if errors.Is(err, ErrSendFailed) {
		t.Fatal("database error was reported as a send failure")
	}
	if sender.calls != 0 {
		t.Fatalf("sender calls: got %d", sender.calls)
	}
	if repo.notifications[0].Status != domain.StatusPending || repo.notifications[0].SendAttempts != 3 {
		t.Fatalf("stored: status=%q attempts=%d", repo.notifications[0].Status, repo.notifications[0].SendAttempts)
	}
}

func TestSendNotificationNotFound(t *testing.T) {
	sender := &fakeEmailSender{}
	uc := newSendUseCase(t, &fakeNotificationRepository{}, sender, 3)

	err := uc.Execute(context.Background(), SendNotificationInput{
		NotificationID:  "missing",
		DeliveryVersion: 1,
	})
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
	uc := newSendUseCase(t, &fakeNotificationRepository{err: repoErr}, sender, 3)

	err := uc.Execute(context.Background(), SendNotificationInput{
		NotificationID:  pendingNotification().ID,
		DeliveryVersion: 1,
	})
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
	notification := pendingNotification()
	notification.SendAttempts = 1
	repo := &fakeNotificationRepository{notifications: []domain.Notification{notification}}
	sender := &fakeEmailSender{}
	uc := newSendUseCase(t, repo, sender, 3)

	err := uc.Execute(context.Background(), SendNotificationInput{
		NotificationID:  notification.ID,
		DeliveryVersion: 1,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if sender.calls != 1 {
		t.Fatalf("sender calls: got %d", sender.calls)
	}
	stored := repo.notifications[0]
	if stored.Status != domain.StatusSent || stored.SendAttempts != 2 || !stored.ClaimedUntil.IsZero() {
		t.Fatalf("stored: %+v", stored)
	}
}

func TestSendNotificationSendFailurePersistsAttempt(t *testing.T) {
	sendErr := errors.New("smtp down")
	repo := &fakeNotificationRepository{notifications: []domain.Notification{pendingNotification()}}
	sender := &fakeEmailSender{err: sendErr}
	uc := newSendUseCase(t, repo, sender, 3)

	err := uc.Execute(context.Background(), SendNotificationInput{
		NotificationID:  pendingNotification().ID,
		DeliveryVersion: 1,
	})
	if !errors.Is(err, ErrSendFailed) || !errors.Is(err, sendErr) {
		t.Fatalf("error: got %v", err)
	}
	if sender.calls != 1 {
		t.Fatalf("sender calls: got %d", sender.calls)
	}
	stored := repo.notifications[0]
	if stored.Status != domain.StatusPending || stored.SendAttempts != 1 || !stored.ClaimedUntil.IsZero() {
		t.Fatalf("stored: %+v", stored)
	}
}

func TestSendNotificationSendFailureUpdateError(t *testing.T) {
	sendErr := errors.New("smtp down")
	repoErr := errors.New("mongo down")
	repo := &fakeNotificationRepository{
		notifications: []domain.Notification{pendingNotification()},
		updateErr:     repoErr,
	}
	sender := &fakeEmailSender{err: sendErr}
	uc := newSendUseCase(t, repo, sender, 3)

	err := uc.Execute(context.Background(), SendNotificationInput{
		NotificationID:  pendingNotification().ID,
		DeliveryVersion: 1,
	})
	if !errors.Is(err, repoErr) {
		t.Fatalf("error: got %v", err)
	}
	if errors.Is(err, ErrSendFailed) {
		t.Fatal("database error replaced the persisted attempt")
	}
	if sender.calls != 1 {
		t.Fatalf("sender calls: got %d", sender.calls)
	}
	stored := repo.notifications[0]
	if stored.Status != domain.StatusPending || stored.SendAttempts != 0 {
		t.Fatalf("stored: status=%q attempts=%d", stored.Status, stored.SendAttempts)
	}
}

func TestSendNotificationFailureAtCeilingAcks(t *testing.T) {
	notification := pendingNotification()
	notification.SendAttempts = 2
	repo := &fakeNotificationRepository{notifications: []domain.Notification{notification}}
	sender := &fakeEmailSender{err: errors.New("smtp down")}
	uc := newSendUseCase(t, repo, sender, 3)

	err := uc.Execute(context.Background(), SendNotificationInput{
		NotificationID:  notification.ID,
		DeliveryVersion: 1,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if sender.calls != 1 {
		t.Fatalf("sender calls: got %d", sender.calls)
	}
	stored := repo.notifications[0]
	if stored.Status != domain.StatusFailed || stored.SendAttempts != 3 {
		t.Fatalf("stored: status=%q attempts=%d", stored.Status, stored.SendAttempts)
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
			notification.SendAttempts = 9
			notification.DeliveryVersion = 4
			repo := &fakeNotificationRepository{notifications: []domain.Notification{notification}}
			sender := &fakeEmailSender{}
			uc := newSendUseCase(t, repo, sender, 3)

			err := uc.Execute(context.Background(), SendNotificationInput{
				NotificationID:  notification.ID,
				DeliveryVersion: 1,
			})
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if sender.calls != 0 {
				t.Fatalf("sender calls: got %d", sender.calls)
			}
			if repo.updated != nil {
				t.Fatal("terminal notification was updated")
			}
			stored := repo.notifications[0]
			if stored.Status != tt.status || stored.SendAttempts != 9 {
				t.Fatalf("stored: %+v", stored)
			}
		})
	}
}

func TestSendNotificationStaleVersionDoesNotSend(t *testing.T) {
	notification := pendingNotification()
	notification.DeliveryVersion = 3
	notification.SendAttempts = 1
	repo := &fakeNotificationRepository{notifications: []domain.Notification{notification}}
	sender := &fakeEmailSender{}
	uc := newSendUseCase(t, repo, sender, 3)

	err := uc.Execute(context.Background(), SendNotificationInput{
		NotificationID:  notification.ID,
		DeliveryVersion: 2,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if sender.calls != 0 {
		t.Fatalf("sender calls: got %d", sender.calls)
	}
	if repo.updated != nil {
		t.Fatal("stale notification was updated")
	}
	stored := repo.notifications[0]
	if stored.Status != domain.StatusPending || stored.SendAttempts != 1 || stored.DeliveryVersion != 3 {
		t.Fatalf("stored: %+v", stored)
	}
}

func TestSendNotificationLostClaimDoesNotSend(t *testing.T) {
	repo := &fakeNotificationRepository{notifications: []domain.Notification{pendingNotification()}}
	sender := &fakeEmailSender{}
	uc := newSendUseCase(t, repo, sender, 3)
	var nested error
	sender.hook = func() {
		nested = uc.Execute(context.Background(), SendNotificationInput{
			NotificationID:  pendingNotification().ID,
			DeliveryVersion: 1,
		})
	}

	err := uc.Execute(context.Background(), SendNotificationInput{
		NotificationID:  pendingNotification().ID,
		DeliveryVersion: 1,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !errors.Is(nested, ErrDeliveryInProgress) {
		t.Fatalf("nested error: got %v", nested)
	}
	if sender.calls != 1 {
		t.Fatalf("sender calls: got %d", sender.calls)
	}
	stored := repo.notifications[0]
	if stored.Status != domain.StatusSent || stored.SendAttempts != 1 {
		t.Fatalf("stored: %+v", stored)
	}
}
