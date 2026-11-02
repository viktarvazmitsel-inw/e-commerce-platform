package usecase

import "errors"

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrSendAttemptLimit     = errors.New("send attempt limit exceeded")
	ErrSendFailed           = errors.New("notification send failed")
)
