package usecase

import (
	"authorization/internal/domain/security"
	"errors"
	"fmt"
)

var ErrSameNewPassword = errors.New("new passwrod must be defferent from current one")

type PasswordUpdateInput struct {
	UserID      string
	OldPassword string
	NewPassword string
}

type PasswordUpdateUseCase struct {
	repo         UserRepository
	tokenStorage TokenRepository
}

func NewPasswordUpdateUseCase(r UserRepository, s TokenRepository) *PasswordUpdateUseCase {
	return &PasswordUpdateUseCase{
		repo:         r,
		tokenStorage: s,
	}
}

func (uc *PasswordUpdateUseCase) Execute(input PasswordUpdateInput) error {
	if input.UserID == "" {
		return ErrUserIdNotFound
	}

	if input.NewPassword == input.OldPassword {
		return ErrSameNewPassword
	}

	userEntity, err := uc.repo.GetUserById(input.UserID)
	if err != nil {
		if errors.Is(err, ErrUserIdNotFound) {
			return ErrUserIdNotFound
		}

		return fmt.Errorf("failed to fetch user from database: %w", err)
	}

	if err := userEntity.EnsureActive(); err != nil {
		return err
	}

	if isPasswordValid, err := security.IsPasswordValid(input.OldPassword, userEntity.PasswordHash); err != nil || !isPasswordValid {
		return ErrInvalidPassword
	}

	if !security.IsPasswordStrong(input.NewPassword) {
		return ErrPasswordIsNotStrongEnough
	}

	newPasswordHash, err := security.HashPassword(input.NewPassword)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPasswordHashFailed, err)
	}

	if err := userEntity.UpdateUserPassword(newPasswordHash); err != nil {
		return err
	}

	if err := uc.repo.Save(userEntity); err != nil {
		return fmt.Errorf("unable to save user: %w", err)
	}

	if err := uc.tokenStorage.RevokeAllUserSessions(input.UserID); err != nil {
		return fmt.Errorf("failed to revoke user session: %w", err)
	}

	return nil

}
