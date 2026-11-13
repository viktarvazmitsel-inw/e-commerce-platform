package handler

import (
	"context"
	"encoding/json"
	"log"

	"github.com/shop/services/notification/internal/infrastructure/broker"
	"github.com/shop/services/notification/internal/usecase"
)

type SendNotificationUseCase interface {
	Execute(ctx context.Context, input usecase.SendNotificationInput) (usecase.SendNotificationOutput, error)
}

type SendVerificationEmailHandler struct {
	uc SendNotificationUseCase
	br *broker.Broker
}

func NewSendVerificationEmailHandler(uc SendNotificationUseCase, br *broker.Broker) *SendVerificationEmailHandler {
	return &SendVerificationEmailHandler{uc: uc, br: br}
}

func (h *SendVerificationEmailHandler) Handle(ctx context.Context, body []byte, metadata broker.MessageMetadata) {
	var payload struct {
		Type  string         `json:"type"`
		Email string         `json:"email"`
		Token map[string]any `json:"token,omitempty"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("failed to unmarshal body: %v", err)
		return
	}

	input := usecase.SendNotificationInput{
		NotificationID: metadata.OriginalID,
		Event:          metadata.RoutingKey,
		Receiver:       payload.Email,
		Payload:        payload.Token,
	}

	output, err := h.uc.Execute(ctx, input)
	if err != nil {
		metadata.RetryCount = output.ResendCount
		metadata.OriginalID = output.NotificationID
		handleError(ctx, h.br, err, body, metadata)
		return
	}
}
