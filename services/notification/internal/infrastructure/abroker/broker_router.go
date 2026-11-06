package abroker

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type MessageHandler func(ctx context.Context, body []byte, resendAttempts int) error

type BrokerRouter struct {
	Services          []string
	HandlerMap        map[string]MessageHandler
	DeadLetterHandler MessageHandler
}

func NewBrokerRouter(
	services []string,
	handlerMap map[string]MessageHandler,
	deadLetterHandler ...MessageHandler,
) *BrokerRouter {
	if len(deadLetterHandler) == 0 {
		deadLetterHandler = []MessageHandler{
			func(ctx context.Context, body []byte, resendAttempts int) error {
				return nil
			},
		}
	}

	return &BrokerRouter{
		Services:          services,
		HandlerMap:        handlerMap,
		DeadLetterHandler: deadLetterHandler[0],
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

	for _, key := range keys {
		if !matchesDeclaredService(router.Services, key) {
			return fmt.Errorf("handler key %q does not match {service}.# for a declared service", key)
		}
	}
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
