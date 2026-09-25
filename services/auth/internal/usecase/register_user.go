package usecase

import (
	"authorization/internal/domain"
	"authorization/internal/domain/security"
	"context"
	"errors"
	"fmt"
	"time"
)

type RegisterInput struct {
	Email    string
	Password string
	Name     string
	Surname  string
	Phone    string
}

type RegisterUserUseCase struct {
	repo           UserRepository
	mail           EmailSender
	tokenStorage   VerificationTokenRepository
	passwordHasher PasswordHasher
}

func NewRegisterUserUseCase(
	r UserRepository,
	m EmailSender,
	s VerificationTokenRepository,
	h PasswordHasher,
	ttl time.Duration,
) *RegisterUserUseCase {
	return &RegisterUserUseCase{
		repo:           r,
		mail:           m,
		tokenStorage:   s,
		passwordHasher: h,
	}
}

func (uc *RegisterUserUseCase) Execute(ctx context.Context, input RegisterInput) error {
	if !security.IsPasswordStrong(input.Password) {
		return ErrPasswordIsNotStrongEnough
	}

	passwordHash, err := uc.passwordHasher.Hash(input.Password)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPasswordHashFailed, err)
	}

	isTaken, err := uc.repo.IsEmailTaken(ctx, input.Email)
	if err != nil {
		return fmt.Errorf("unable to connect database: %w", err)
	}

	if isTaken {
		return ErrEmailAlreadyTaken
	}

	userEntity, err := domain.NewUser(input.Email, passwordHash, input.Name, input.Surname, input.Phone)
	if err != nil {
		return err
	}

	if err := uc.repo.Save(ctx, userEntity); err != nil {
		if errors.Is(err, ErrEmailAlreadyTaken) {
			return ErrEmailAlreadyTaken
		}
		return fmt.Errorf("unable to save user: %w", err)
	}

	token, err := security.GenerateUserDataUpdateToken()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTokenGenerationFailed, err)
	}

	if err := uc.tokenStorage.SaveVerificationToken(ctx, userEntity.ID, token); err != nil {
		return fmt.Errorf("%w: %v", ErrVerificationTokenSaveFailed, err)
	}

	if err := uc.mail.SendVerificationEmail(ctx, userEntity.Email, token); err != nil {
		return fmt.Errorf("%w: %v", ErrVerificationEmailSendFailed, err)
	}

	return nil
}
