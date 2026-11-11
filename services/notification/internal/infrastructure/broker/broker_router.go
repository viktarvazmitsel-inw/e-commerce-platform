package broker

import (
	"context"
)

type MessageMetadata struct {
	OriginalID string
	RetryCount int
	RoutingKey string
}

type MessageHandler func(ctx context.Context, body []byte, metadata MessageMetadata)

type BrokerRouter struct {
	HandlerMap map[string]MessageHandler
}

func NewBrokerRouter(
	handlerMap map[string]MessageHandler,
) *BrokerRouter {

	return &BrokerRouter{
		HandlerMap: handlerMap,
	}
}
