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

type PasswordUpdateUseCase struct {
	repo           UserRepository
	tokenStorage   SessionRepository
	passwordHasher PasswordHasher
}

func NewPasswordUpdateUseCase(r UserRepository, s SessionRepository, h PasswordHasher) *PasswordUpdateUseCase {
	return &PasswordUpdateUseCase{
		repo:           r,
		tokenStorage:   s,
		passwordHasher: h,
	}
}

func (uc *PasswordUpdateUseCase) Execute(ctx context.Context, input PasswordUpdateInput) error {
	if input.UserID == "" {
		return ErrUserIdNotFound
	}

	if input.NewPassword == input.OldPassword {
		return ErrSameNewPassword
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

	if !uc.passwordHasher.Compare(input.OldPassword, userEntity.PasswordHash) {
		return ErrInvalidPassword
	}

	if !security.IsPasswordStrong(input.NewPassword) {
		return ErrPasswordIsNotStrongEnough
	}

	newPasswordHash, err := uc.passwordHasher.Hash(input.NewPassword)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPasswordHashFailed, err)
	}

	if err := userEntity.UpdateUserPassword(newPasswordHash); err != nil {
		return err
	}

	if err := uc.repo.Save(ctx, userEntity); err != nil {
		return fmt.Errorf("unable to save user: %w", err)
	}

	if err := uc.tokenStorage.RevokeAllUserSessions(ctx, input.UserID); err != nil {
		return fmt.Errorf("failed to revoke user session: %w", err)
	}

	return nil

}
