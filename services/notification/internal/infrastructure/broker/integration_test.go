package broker

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/shop/services/notification/internal/usecase"
)

type fakeCreator struct {
	mu     sync.Mutex
	inputs []usecase.CreateNotificationInput
}

func (f *fakeCreator) Execute(ctx context.Context, input usecase.CreateNotificationInput) (string, error) {
	f.mu.Lock()
	f.inputs = append(f.inputs, input)
	f.mu.Unlock()
	return "notif-1", nil
}

func (f *fakeCreator) snapshot() []usecase.CreateNotificationInput {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]usecase.CreateNotificationInput, len(f.inputs))
	copy(out, f.inputs)
	return out
}

type fakeSender struct {
	mu    sync.Mutex
	calls []usecase.SendNotificationInput
}

func (f *fakeSender) Execute(ctx context.Context, input usecase.SendNotificationInput) error {
	f.mu.Lock()
	f.calls = append(f.calls, input)
	f.mu.Unlock()
	return nil
}

func (f *fakeSender) snapshot() []usecase.SendNotificationInput {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]usecase.SendNotificationInput, len(f.calls))
	copy(out, f.calls)
	return out
}

func TestBrokerIntegration(t *testing.T) {
	url := os.Getenv("NOTIFICATION_BROKER_URL")
	if url == "" {
		t.Skip("NOTIFICATION_BROKER_URL is not set")
	}

	creator := &fakeCreator{}
	sender := &fakeSender{}
	b, err := New(url)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(b.Close)
	purgeQueues(t, b)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- b.Run(ctx, creator, sender)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-errCh:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Errorf("Run: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("Run did not stop")
		}
	})

	pub, err := amqp.Dial(url)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = pub.Close() })
	ch, err := pub.Channel()
	if err != nil {
		t.Fatalf("channel: %v", err)
	}

	body := []byte(`{"type":"user.created","email":"user@example.com","token":"abc"}`)
	if err := ch.PublishWithContext(ctx, emailExchange, authCreatedKey, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	}); err != nil {
		t.Fatalf("publish auth event: %v", err)
	}

	waitFor(t, func() bool {
		return len(creator.snapshot()) == 1
	})
	got := creator.snapshot()[0]
	want := usecase.CreateNotificationInput{
		Channel:  "email",
		Receiver: "user@example.com",
		Payload:  `{"type":"user.created","token":"abc"}`,
	}
	if got != want {
		t.Fatalf("create input: got %+v", got)
	}

	if err := b.PublishForDelivery(ctx, "notif-1", 2); err != nil {
		t.Fatalf("PublishForDelivery: %v", err)
	}
	waitFor(t, func() bool {
		return len(sender.snapshot()) == 1
	})
	sent := sender.snapshot()[0]
	if sent.NotificationID != "notif-1" || sent.DeliveryVersion != 2 {
		t.Fatalf("delivery: %+v", sent)
	}

	poison := []byte(`{`)
	if err := ch.PublishWithContext(ctx, emailExchange, authCreatedKey, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        poison,
	}); err != nil {
		t.Fatalf("publish poison: %v", err)
	}

	dlq, err := ch.Consume(emailDLQ, "", true, false, false, false, nil)
	if err != nil {
		t.Fatalf("consume dlq: %v", err)
	}
	select {
	case d := <-dlq:
		if string(d.Body) != string(poison) {
			t.Fatalf("dlq body: %s", d.Body)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("poison message was not dead-lettered")
	}
	if len(creator.snapshot()) != 1 {
		t.Fatalf("creator calls: got %d", len(creator.snapshot()))
	}
}

func purgeQueues(t *testing.T, b *Broker) {
	t.Helper()
	for _, queue := range []string{emailQueue, emailDLQ, deliveryQueue, deliveryDLQ} {
		if _, err := b.publishCh.QueuePurge(queue, false); err != nil {
			t.Fatalf("purge %s: %v", queue, err)
		}
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timed out waiting for broker")
}
