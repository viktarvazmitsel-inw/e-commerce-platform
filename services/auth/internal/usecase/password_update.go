package usecase

import (
	"authorization/internal/domain/security"
	"context"
	"errors"
	"fmt"
)

type PasswordUpdateInput struct {
	UserID      string
	OldPassword string
	NewPassword string
}

type PasswordUpdateResult struct {
	UserID         string
	SessionRevoked bool
}

type PasswordUpdateUseCase struct {
	repo           UserRepository
	tokenStorage   SessionRepository
	passwordHasher PasswordHasher
	logger         Logger
}

func NewPasswordUpdateUseCase(
	r UserRepository,
	s SessionRepository,
	h PasswordHasher,
	l Logger,
) *PasswordUpdateUseCase {
	return &PasswordUpdateUseCase{
		repo:           r,
		tokenStorage:   s,
		passwordHasher: h,
		logger:         l,
	}
}

func (uc *PasswordUpdateUseCase) Execute(ctx context.Context, input PasswordUpdateInput) (PasswordUpdateResult, error) {
	if input.UserID == "" {
		return PasswordUpdateResult{}, ErrUserIdNotFound
	}

	if input.NewPassword == input.OldPassword {
		return PasswordUpdateResult{}, ErrSameNewPassword
	}

	userEntity, err := uc.repo.GetUserByID(ctx, input.UserID)
	if err != nil {
		if errors.Is(err, ErrUserIdNotFound) {
			return PasswordUpdateResult{}, ErrUserIdNotFound
		}

		return PasswordUpdateResult{}, fmt.Errorf("failed to fetch user from database: %w", err)
	}

	if err := userEntity.EnsureActive(); err != nil {
		return PasswordUpdateResult{}, err
	}

	if !uc.passwordHasher.Compare(input.OldPassword, userEntity.PasswordHash) {
		return PasswordUpdateResult{}, ErrInvalidPassword
	}

	if !security.IsPasswordStrong(input.NewPassword) {
		return PasswordUpdateResult{}, ErrPasswordIsNotStrongEnough
	}

	newPasswordHash, err := uc.passwordHasher.Hash(input.NewPassword)
	if err != nil {
		return PasswordUpdateResult{}, fmt.Errorf("%w: %v", ErrPasswordHashFail, err)
	}

	if err := userEntity.UpdateUserPassword(newPasswordHash); err != nil {
		return PasswordUpdateResult{}, err
	}

	if err := uc.repo.Save(ctx, userEntity); err != nil {
		return PasswordUpdateResult{}, fmt.Errorf("unable to save user: %w", err)
	}

	if err := uc.tokenStorage.RevokeAllUserSessions(ctx, input.UserID); err != nil {
		uc.logger.Warn(ctx, "failed to revoke user sessions", "userID", userEntity.ID, "error", err)
		return PasswordUpdateResult{UserID: userEntity.ID, SessionRevoked: false}, nil
	}

	return PasswordUpdateResult{UserID: userEntity.ID, SessionRevoked: true}, nil

}
