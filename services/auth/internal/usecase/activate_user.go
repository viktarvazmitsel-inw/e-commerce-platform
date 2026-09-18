package usecase

import (
	"errors"
	"fmt"
)

var (
	ErrWrongVerificationToken      = errors.New("wrong verification token")
	ErrUserActivationFail          = errors.New("user activation failed")
	ErrVerificationTokenDeleteFail = errors.New("failed to delete verification token")
)

type VerificationInput struct {
	Token string
}

type ActivateUserUseCase struct {
	repo         UserRepository
	tokenStorage TokenRepository
}

func NewActivateUserUseCase(r UserRepository, s TokenRepository) *ActivateUserUseCase {
	return &ActivateUserUseCase{
		repo:         r,
		tokenStorage: s,
	}
}

func (uc *ActivateUserUseCase) Execute(input VerificationInput) error {
	userToken := input.Token

	userId, err := uc.tokenStorage.LoadVerificationToken(userToken)
	if err != nil {
		return ErrWrongVerificationToken
	}

	if err := uc.repo.ActivateUser(userId); err != nil {
		return fmt.Errorf("%w: %v", ErrUserActivationFail, err)
	}

	if err := uc.tokenStorage.DeleteVerificationToken(userToken); err != nil {
		return ErrVerificationTokenDeleteFail
	}

	return nil
}
