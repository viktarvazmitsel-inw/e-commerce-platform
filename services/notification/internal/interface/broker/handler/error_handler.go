package handler

import (
	"context"
	"errors"
	"log"

	"github.com/shop/services/notification/internal/infrastructure/broker"
	"github.com/shop/services/notification/internal/infrastructure/email"
	"github.com/shop/services/notification/internal/usecase"
)

func handleError(
	ctx context.Context,
	broker *broker.Broker,
	err error,
	body []byte,
	metadata broker.MessageMetadata,
) {
	switch {
	case errors.Is(err, usecase.ErrMaxResendAttemptsReached):
		err := broker.SendToDeadLetterQueue(ctx, body, metadata)
		if err != nil {
			log.Printf("failed to send message to dead letter queue: %v", err)
			return
		}
		log.Printf("message sent to dead letter queue: %v", metadata.OriginalID)
		return
	case errors.Is(err, email.ErrUnableToSendEmail):
		err := broker.ResendMessage(ctx, body, metadata)
		if err != nil {
			log.Printf("failed to resend message: %v", err)
			return
		}
		log.Printf("message resent: %v", metadata.OriginalID)
		return
	default:
		log.Printf("failed to send notification: %v", err)
	}
}
