package broker

import (
	"context"
	"errors"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestRetryCountFromHeader(t *testing.T) {
	tests := []struct {
		name    string
		headers amqp.Table
		want    int
	}{
		{name: "missing", headers: amqp.Table{}, want: 0},
		{name: "int32", headers: amqp.Table{"x-retry-count": int32(2)}, want: 2},
		{name: "int", headers: amqp.Table{"x-retry-count": 1}, want: 1},
		{name: "int64", headers: amqp.Table{"x-retry-count": int64(3)}, want: 3},
		{name: "string", headers: amqp.Table{"x-retry-count": "2"}, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := retryCountFromHeader(tt.headers); got != tt.want {
				t.Fatalf("retryCountFromHeader() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCloseUnblocksConsumerBeforeWaiting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	broker := &Broker{}
	broker.setCancel(cancel)

	broker.wg.Add(1)
	finished := make(chan struct{})
	go func() {
		defer broker.wg.Done()
		<-ctx.Done()
		close(finished)
	}()

	if err := broker.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-finished:
	default:
		t.Fatal("consumer was still running after Close")
	}
}

func TestPublishReturnsStoredConfirmError(t *testing.T) {
	broker := &Broker{publishErr: errors.New("publish confirm interrupted")}
	err := broker.publish(context.Background(), "", deadLetterQueue, amqp.Publishing{})
	if !errors.Is(err, broker.publishErr) {
		t.Fatalf("publish() error = %v", err)
	}
}

func TestReturnedPublish(t *testing.T) {
	broker := &Broker{returns: make(chan amqp.Return, 1)}

	if _, ok := broker.returnedPublish(); ok {
		t.Fatal("expected no return")
	}

	broker.returns <- amqp.Return{Exchange: retryExchange, RoutingKey: "notification.email", ReplyText: "NO_ROUTE"}
	ret, ok := broker.returnedPublish()
	if !ok {
		t.Fatal("expected return")
	}
	if ret.ReplyText != "NO_ROUTE" || ret.RoutingKey != "notification.email" {
		t.Fatalf("unexpected return: %+v", ret)
	}
	if _, ok := broker.returnedPublish(); ok {
		t.Fatal("expected return to be consumed")
	}
}
