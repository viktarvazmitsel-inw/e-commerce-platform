package usecase

import (
	"authorization/internal/domain/security"
	"context"
	"errors"
	"fmt"
	"time"
)

type ResendVerificationEmailInput struct {
	UserID   string
	Password string
	tokenTTL time.Duration
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
	if input.UserID == "" {
		return ErrUserIdNotFound
	}

	userEntity, err := uc.repo.GetUserByID(ctx, input.UserID)
	if err != nil {
		return ErrUserIdNotFound
	}

	if !uc.passwordHasher.Compare(input.Password, userEntity.PasswordHash) {
		return ErrInvalidPassword
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

	if err := uc.emailTokenStorage.SaveVerificationToken(ctx, input.UserID, token, input.tokenTTL); err != nil {
		if errors.Is(err, ErrVerificationTokenSaveFailed) {
			return ErrVerificationTokenSaveFailed
		}

		return fmt.Errorf("unable to save verification token: %w", err)
	}

	if err := uc.mail.SendVerificationEmail(ctx, userEntity.Email, token); err != nil {
		return fmt.Errorf("failed to send verification email: %w", ErrVerificationEmailSendFailed)
	}

	return nil
}
