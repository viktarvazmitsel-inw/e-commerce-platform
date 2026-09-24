package usecase

import (
	"context"
	"fmt"
)

type VerificationInput struct {
	Token string
}

type VerifyUserUseCase struct {
	repo         UserRepository
	tokenStorage VerificationTokenRepository
}

func NewVerifyUserUseCase(r UserRepository, s VerificationTokenRepository) *VerifyUserUseCase {
	return &VerifyUserUseCase{
		repo:         r,
		tokenStorage: s,
	}
}

func (uc *VerifyUserUseCase) Execute(ctx context.Context, input VerificationInput) error {
	userToken := input.Token

	userID, err := uc.tokenStorage.LoadVerificationToken(ctx, userToken)
	if err != nil {
		return ErrWrongVerificationToken
	}

	userEntity, err := uc.repo.GetUserByID(ctx, userID)
	if err != nil {
		return ErrUserIdNotFound
	}

	userEntity.VerifyEmail()

	if err := uc.repo.Save(ctx, userEntity); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	if err := uc.tokenStorage.DeleteVerificationToken(ctx, userToken); err != nil {
		return ErrVerificationTokenDeleteFail
	}

	return nil
}
