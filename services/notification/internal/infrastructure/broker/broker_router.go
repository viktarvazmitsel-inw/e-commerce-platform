package broker

import (
	"context"
	"errors"
	"sort"
	"strings"
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
	services []string,
	handlerMap map[string]MessageHandler,
) *BrokerRouter {

	return &BrokerRouter{
		HandlerMap: handlerMap,
	}
}

func validateRouter(router *BrokerRouter) error {
	if router == nil {
		return errors.New("broker router is required")
	}

	keys := make([]string, 0, len(router.HandlerMap))
	for key := range router.HandlerMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return nil
}

func matchesDeclaredService(services []string, routingKey string) bool {
	for _, service := range services {
		if matchesServiceBinding(service, routingKey) {
			return true
		}
	}
	return false
}

// matchesServiceBinding reports whether routingKey matches the topic binding service.#.
func matchesServiceBinding(service, routingKey string) bool {
	if service == "" || routingKey == "" {
		return false
	}
	return routingKey == service || strings.HasPrefix(routingKey, service+".")
}
