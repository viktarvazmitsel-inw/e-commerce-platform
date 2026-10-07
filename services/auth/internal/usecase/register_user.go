package usecase

import (
	"authorization/internal/domain"
	"authorization/internal/domain/security"
	"context"
	"errors"
	"fmt"
)

type RegisterInput struct {
	Email    string
	Password string
	Name     string
	Surname  string
	Phone    string
}

type RegisterUserResult struct {
	UserID           string
	VerificationSent bool
}

type RegisterUserUseCase struct {
	repo           UserRepository
	mail           EmailSender
	tokenStorage   VerificationTokenRepository
	passwordHasher PasswordHasher
	logger         Logger
}

func NewRegisterUserUseCase(
	r UserRepository,
	m EmailSender,
	s VerificationTokenRepository,
	h PasswordHasher,
	l Logger,
) *RegisterUserUseCase {
	return &RegisterUserUseCase{
		repo:           r,
		mail:           m,
		tokenStorage:   s,
		passwordHasher: h,
		logger:         l,
	}
}

func (uc *RegisterUserUseCase) Execute(ctx context.Context, input RegisterInput) (RegisterUserResult, error) {
	if !security.IsPasswordStrong(input.Password) {
		return RegisterUserResult{}, ErrPasswordIsNotStrongEnough
	}

	passwordHash, err := uc.passwordHasher.Hash(input.Password)
	if err != nil {
		return RegisterUserResult{}, fmt.Errorf("%w: %v", ErrPasswordHashFail, err)
	}

	isTaken, err := uc.repo.IsEmailTaken(ctx, input.Email)
	if err != nil {
		return RegisterUserResult{}, fmt.Errorf("unable to connect database: %w", err)
	}

	if isTaken {
		return RegisterUserResult{}, ErrEmailAlreadyTaken
	}

	userEntity, err := domain.NewUser(input.Email, passwordHash, input.Name, input.Surname, input.Phone)
	if err != nil {
		return RegisterUserResult{}, err
	}

	if err := uc.repo.Create(ctx, userEntity); err != nil {
		if errors.Is(err, ErrEmailAlreadyTaken) {
			return RegisterUserResult{}, ErrEmailAlreadyTaken
		}
		return RegisterUserResult{}, fmt.Errorf("failed to save user: %w", err)
	}

	token, err := security.GenerateUserDataUpdateToken()
	if err != nil {
		uc.logger.Warn(ctx, "failed to generate email verification token", "userID", userEntity.ID, "error", err)
		return RegisterUserResult{UserID: userEntity.ID, VerificationSent: false}, nil
	}

	hashedToken := security.HashToken(token)

	if err := uc.tokenStorage.SaveVerificationToken(ctx, userEntity.ID, hashedToken); err != nil {
		uc.logger.Warn(ctx, "failed to save email verification token", "userID", userEntity.ID, "error", err)
		return RegisterUserResult{UserID: userEntity.ID, VerificationSent: false}, nil
	}

	if err := uc.mail.SendVerificationEmail(ctx, userEntity.Email, token); err != nil {
		uc.logger.Warn(ctx, "failed to send verification email to user", "userID", userEntity.ID, "error", err)
		return RegisterUserResult{UserID: userEntity.ID, VerificationSent: false}, nil
	}

	return RegisterUserResult{UserID: userEntity.ID, VerificationSent: true}, nil
}
