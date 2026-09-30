package domain

import "errors"

var (
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrEmptyEmail       = errors.New("email can not be empty")
	ErrEmptyPassword    = errors.New("password cannot be empty")
	ErrEmptyName        = errors.New("name cannot be empty")
	ErrEmptySurname     = errors.New("surname cannot be empty")
	ErrInvalidPhone     = errors.New("invalid phone format")
	ErrPermissionDenied = errors.New("permission denied")
	ErrUnableToSetAdmin = errors.New("unable to set admin role")
	ErrUnchangedRole    = errors.New("role didn't change")
	ErrUserDeactivated  = errors.New("profile deleted")
)
