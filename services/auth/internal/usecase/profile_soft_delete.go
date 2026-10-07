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

type SoftDeleteProfileResult struct {
	UserID         string
	SessionRevoked bool
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

func (uc *SoftDeleteProfileUseCase) Execute(ctx context.Context, input SoftDeleteProfileInput) (SoftDeleteProfileResult, error) {
	if input.UserID == "" {
		return SoftDeleteProfileResult{}, ErrUserIdNotFound
	}

	userEntity, err := uc.repo.GetUserByID(ctx, input.UserID)
	if err != nil {
		if errors.Is(err, ErrUserIdNotFound) {
			return SoftDeleteProfileResult{}, ErrUserIdNotFound
		}

		return SoftDeleteProfileResult{}, fmt.Errorf("failed to fetch user from database: %w", err)
	}

	if err := userEntity.EnsureActive(); err != nil {
		return SoftDeleteProfileResult{}, err
	}

	if !uc.passwordHasher.Compare(input.Password, userEntity.PasswordHash) {
		return SoftDeleteProfileResult{}, ErrInvalidPassword
	}

	userEntity.Deactivate()

	if err := uc.repo.Save(ctx, userEntity); err != nil {
		return SoftDeleteProfileResult{}, fmt.Errorf("unable to save user: %w", err)
	}

	if err := uc.tokenStorage.RevokeAllUserSessions(ctx, userEntity.ID); err != nil {
		uc.logger.Warn(ctx, "failed to revoke user session", "userID", userEntity.ID, "error", err)
		return SoftDeleteProfileResult{UserID: userEntity.ID, SessionRevoked: false}, nil
	}

	return SoftDeleteProfileResult{UserID: userEntity.ID, SessionRevoked: true}, nil
}
