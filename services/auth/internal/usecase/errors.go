package usecase

import "errors"

var (
	ErrTokenPairGenerationFail = errors.New("unable to generate access and refresh tokens")
	ErrTokenPairSaveFail       = errors.New("unable to save access and refresh tokens")
	ErrUserIdNotFound          = errors.New("user with such id not found")
)
