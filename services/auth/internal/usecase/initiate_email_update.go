package usecase

import (
	"authorization/internal/domain"
	"authorization/internal/domain/security"
	"context"
	"errors"
	"fmt"
	"strings"
)

type EmailUpdateInput struct {
	ID       string
	Password string
	NewEmail string
}

type InitiateEmailUpdateUseCase struct {
	repo           UserRepository
	mail           EmailSender
	tokenStorage   UserDataUpdateTokenRepository
	passwordHasher PasswordHasher
}

func NewInitiateEmailUpdateUseCase(
	r UserRepository,
	m EmailSender,
	s UserDataUpdateTokenRepository,
	h PasswordHasher,
) *InitiateEmailUpdateUseCase {
	return &InitiateEmailUpdateUseCase{
		repo:           r,
		mail:           m,
		tokenStorage:   s,
		passwordHasher: h,
	}
}

func (uc *InitiateEmailUpdateUseCase) Execute(ctx context.Context, input EmailUpdateInput) error {
	if input.ID == "" {
		return ErrUserIdNotFound
	}

	userEntity, err := uc.repo.GetUserByID(ctx, input.ID)
	if err != nil {
		if errors.Is(err, ErrUserIdNotFound) {
			return ErrUserIdNotFound
		}
		return fmt.Errorf("failed to fetch user from database: %w", err)
	}

	if err := userEntity.EnsureActive(); err != nil {
		return err
	}

	if !uc.passwordHasher.Compare(input.Password, userEntity.PasswordHash) {
		return ErrInvalidPassword
	}

	trimmedNewEmail := strings.TrimSpace(input.NewEmail)

	if trimmedNewEmail == userEntity.Email {
		return ErrSameEmail
	}

	if !domain.IsEmailValid(trimmedNewEmail) {
		return domain.ErrInvalidEmail
	}

	isTaken, err := uc.repo.IsEmailTaken(ctx, trimmedNewEmail)
	if err != nil {
		return fmt.Errorf("unable to connect database: %w", err)
	}

	if isTaken {
		return ErrEmailAlreadyTaken
	}

	token, err := security.GenerateUserDataUpdateToken()
	if err != nil {
		return ErrTokenGenerationFail
	}

	hashedToken := security.HashToken(token)

	if err := uc.tokenStorage.SaveEmailUpdateToken(ctx, userEntity.ID, trimmedNewEmail, hashedToken); err != nil {
		if errors.Is(err, ErrVerificationTokenSaveFail) {
			return ErrVerificationTokenSaveFail
		}

		return fmt.Errorf("failed to save email update token: %w", err)
	}

	if err := uc.mail.SendVerificationEmail(ctx, trimmedNewEmail, token); err != nil {
		return ErrVerificationEmailSendFail
	}

	return nil
}
