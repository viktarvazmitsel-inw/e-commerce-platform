package usecase

import (
	"context"
	"errors"
	"strings"

	"authorization/internal/domain/security"
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
			return ErrUserEmailNotFound
		}
		return err
	}

	if err := userEntity.EnsureActive(); err != nil {
		return err
	}

	resetToken, err := security.GenerateUserDataUpdateToken()
	if err != nil {
		return ErrTokenGenerationFail
	}

	hashedResetToken := security.HashToken(resetToken)

	if err := uc.tokenStorage.SavePasswordResetToken(ctx, userEntity.ID, hashedResetToken); err != nil {
		return ErrVerificationTokenSaveFail
	}

	if err := uc.mail.SendPasswordResetEmail(ctx, userEntity.Email, resetToken); err != nil {
		return ErrPasswordResetEmailSendFail
	}

	return nil
}
