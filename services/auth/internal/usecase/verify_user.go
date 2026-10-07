package usecase

import (
	"authorization/internal/domain/security"
	"context"
	"fmt"
)

type VerificationInput struct {
	Token string
}

type VerifyUserUseCase struct {
	repo         UserRepository
	tokenStorage VerificationTokenRepository
	logger       Logger
}

func NewVerifyUserUseCase(r UserRepository, s VerificationTokenRepository, l Logger) *VerifyUserUseCase {
	return &VerifyUserUseCase{
		repo:         r,
		tokenStorage: s,
		logger:       l,
	}
}

func (uc *VerifyUserUseCase) Execute(ctx context.Context, input VerificationInput) error {
	if input.Token == "" {
		return ErrWrongVerificationToken
	}

	hashedToken := security.HashToken(input.Token)

	userID, err := uc.tokenStorage.LoadVerificationToken(ctx, hashedToken)
	if err != nil {
		return ErrWrongVerificationToken
	}

	userEntity, err := uc.repo.GetUserByID(ctx, userID)
	if err != nil {
		return ErrUserIdNotFound
	}

	if err := userEntity.EnsureActive(); err != nil {
		return err
	}

	userEntity.VerifyEmail()

	if err := uc.repo.Save(ctx, userEntity); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	if err := uc.tokenStorage.DeleteVerificationToken(ctx, hashedToken); err != nil {
		uc.logger.Warn(ctx, "failed to delete email verification token", "userID", userEntity.ID, "error", err)
		return nil
	}

	return nil
}
