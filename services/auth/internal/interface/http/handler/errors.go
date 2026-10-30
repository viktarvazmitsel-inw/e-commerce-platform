package handler

import (
	"errors"
)

var (
	ErrUserIDUndefined = errors.New("user id is undefined")
	ErrUserIDInvalid   = errors.New("user id is invalid")
)
