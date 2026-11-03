package usecase

import "errors"

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrDeliveryInProgress   = errors.New("notification delivery in progress")
	ErrSendFailed           = errors.New("notification send failed")
)
