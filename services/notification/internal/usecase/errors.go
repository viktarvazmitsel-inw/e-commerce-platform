package usecase

import "errors"

var (
	ErrNotificationNotFound     = errors.New("notification not found")
	ErrMaxResendAttemptsReached = errors.New("max resend attempts reached")
)
