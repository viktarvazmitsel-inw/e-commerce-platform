package usecase

import (
	"authorization/internal/domain/security"
	"context"
	"errors"
	"fmt"
	"strings"
)

type ResendVerificationEmailInput struct {
	Email    string
	Password string
}

type ResendVerificationEmailUseCase struct {
	emailTokenStorage VerificationTokenRepository
	repo              UserRepository
	passwordHasher    PasswordHasher
	mail              EmailSender
}

func NewResendVerificationEmailUseCase(
	s VerificationTokenRepository,
	r UserRepository,
	h PasswordHasher,
	m EmailSender,
) *ResendVerificationEmailUseCase {
	return &ResendVerificationEmailUseCase{
		emailTokenStorage: s,
		repo:              r,
		passwordHasher:    h,
		mail:              m,
	}
}

func (uc *ResendVerificationEmailUseCase) Execute(
	ctx context.Context,
	input ResendVerificationEmailInput,
) error {
	trimmedEmail := strings.TrimSpace(input.Email)
	if trimmedEmail == "" {
		return ErrEmptyEmail
	}

	userEntity, err := uc.repo.GetUserByEmail(ctx, trimmedEmail)
	if err != nil {
		return ErrInvalidCredentials
	}

	if !uc.passwordHasher.Compare(input.Password, userEntity.PasswordHash) {
		return ErrInvalidCredentials
	}

	if err := userEntity.EnsureActive(); err != nil {
		return err
	}

	if userEntity.IsVerified {
		return ErrEmailAlreadyVerified
	}

	token, err := security.GenerateUserDataUpdateToken()
	if err != nil {
		return err
	}

	hashedResetToken := security.HashToken(token)

	if err := uc.emailTokenStorage.SaveVerificationToken(ctx, userEntity.ID, hashedResetToken); err != nil {
		if errors.Is(err, ErrVerificationTokenSaveFail) {
			return ErrVerificationTokenSaveFail
		}

		return fmt.Errorf("unable to save verification token: %w", err)
	}

	if err := uc.mail.SendVerificationEmail(ctx, userEntity.Email, token); err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	return nil
}
