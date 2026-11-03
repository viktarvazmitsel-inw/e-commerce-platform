package broker

import (
	"errors"

	"github.com/shop/services/notification/internal/domain"
	"github.com/shop/services/notification/internal/usecase"
)

type disposition int

const (
	ack disposition = iota
	requeue
	deadLetter
)

func settleAuth(notificationID string, err error) disposition {
	if err == nil {
		return ack
	}
	if errors.Is(err, ErrPoison) || isDomainValidation(err) {
		return deadLetter
	}
	if notificationID != "" {
		return ack
	}
	return requeue
}

func settleDelivery(err error) disposition {
	if err == nil || errors.Is(err, usecase.ErrDeliveryInProgress) {
		return ack
	}
	if errors.Is(err, ErrPoison) || errors.Is(err, usecase.ErrNotificationNotFound) {
		return deadLetter
	}
	return requeue
}

func isDomainValidation(err error) bool {
	return errors.Is(err, domain.ErrInvalidChannel) ||
		errors.Is(err, domain.ErrInvalidReceiver) ||
		errors.Is(err, domain.ErrInvalidPayload)
}
