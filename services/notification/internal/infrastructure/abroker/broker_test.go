package abroker

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

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
