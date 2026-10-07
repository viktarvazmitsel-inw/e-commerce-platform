package usecase

import (
	"context"
	"errors"
	"fmt"
)

type SoftDeleteProfileInput struct {
	UserID   string
	Password string
}

type SoftDeleteProfileUseCase struct {
	repo           UserRepository
	tokenStorage   SessionRepository
	passwordHasher PasswordHasher
	logger         Logger
}

func NewSoftDeleteProfileUseCase(
	r UserRepository,
	s SessionRepository,
	h PasswordHasher,
	l Logger,
) *SoftDeleteProfileUseCase {
	return &SoftDeleteProfileUseCase{
		repo:           r,
		tokenStorage:   s,
		passwordHasher: h,
		logger:         l,
	}
}

func (uc *SoftDeleteProfileUseCase) Execute(ctx context.Context, input SoftDeleteProfileInput) error {
	if input.UserID == "" {
		return ErrUserIdNotFound
	}

	userEntity, err := uc.repo.GetUserByID(ctx, input.UserID)
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

	userEntity.Deactivate()

	if err := uc.repo.Save(ctx, userEntity); err != nil {
		return fmt.Errorf("unable to save user: %w", err)
	}

	if err := uc.tokenStorage.RevokeAllUserSessions(ctx, userEntity.ID); err != nil {
		uc.logger.Warn(ctx, "failed to revoke user session", "userID", userEntity.ID, "error", err)
		return nil
	}

	return nil
}
