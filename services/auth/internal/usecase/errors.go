package usecase

import "errors"

var (
	ErrEmailAlreadyTaken           = errors.New("email already taken")
	ErrInvalidPassword             = errors.New("invalid password")
	ErrTokenGenerationFailed       = errors.New("registration failed due to token generation fail")
	ErrVerificationTokenSaveFailed = errors.New("failed to save verification token")
	ErrVerificationEmailSendFailed = errors.New("failed to send verification email")
	ErrTokenPairGenerationFail     = errors.New("unable to generate access and refresh tokens")
	ErrTokenPairSaveFail           = errors.New("unable to save access and refresh tokens")
	ErrUserIdNotFound              = errors.New("user with such id not found")
)
