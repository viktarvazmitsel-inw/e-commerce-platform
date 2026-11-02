package domain

import "errors"

var (
	ErrInvalidChannel          = errors.New("invalid channel")
	ErrInvalidReceiver         = errors.New("invalid receiver")
	ErrInvalidPayload          = errors.New("invalid payload")
	ErrInvalidStatus           = errors.New("invalid status")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
)
