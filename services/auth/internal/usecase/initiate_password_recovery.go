package usecase

import (
	"authorization/internal/domain/security"
	"context"
	"errors"
	"fmt"
	"strings"
)

type InitiatePasswordRecoveryInput struct {
	UserEmail string
}

type InitiatePasswordRecoveryUseCase struct {
	repo         UserRepository
	tokenStorage UserDataUpdateTokenRepository
	mail         EmailSender
}

func NewInitiatePasswordRecoveryUseCase(
	r UserRepository,
	s UserDataUpdateTokenRepository,
	m EmailSender,
) *InitiatePasswordRecoveryUseCase {
	return &InitiatePasswordRecoveryUseCase{
		repo:         r,
		tokenStorage: s,
		mail:         m,
	}
}

func (uc *InitiatePasswordRecoveryUseCase) Execute(ctx context.Context, input InitiatePasswordRecoveryInput) error {
	trimmedEmail := strings.TrimSpace(input.UserEmail)
	if trimmedEmail == "" {
		return ErrEmptyEmail
	}

	userEntity, err := uc.repo.GetUserByEmail(ctx, trimmedEmail)
	if err != nil {
		if errors.Is(err, ErrUserEmailNotFound) {
			return nil
		}

		return fmt.Errorf("unable to load user from database: %w", err)
	}

	if err := userEntity.EnsureActive(); err != nil {
		return nil
	}

	resetToken, err := security.GenerateUserDataUpdateToken()
	if err != nil {
		return ErrTokenGenerationFail
	}

	hashedResetToken := security.HashToken(resetToken)

	if err := uc.tokenStorage.SavePasswordResetToken(ctx, userEntity.ID, hashedResetToken); err != nil {
		return fmt.Errorf("failed to save password reset token: %w", err)
	}

	if err := uc.mail.SendPasswordResetEmail(ctx, userEntity.Email, resetToken); err != nil {
		return ErrPasswordResetEmailSendFail
	}

	return nil
}
