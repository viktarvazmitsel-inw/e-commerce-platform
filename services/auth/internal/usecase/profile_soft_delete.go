package usecase

import (
	"authorization/internal/domain/security"
	"errors"
	"fmt"
	"log"
)

type SoftDeleteProfileInput struct {
	UserID   string
	Password string
}

type SoftDeleteProfileUseCase struct {
	repo         UserRepository
	tokenStorage TokenRepository
}

func NewSoftDeleteProfileUseCase(r UserRepository, s TokenRepository) *SoftDeleteProfileUseCase {
	return &SoftDeleteProfileUseCase{
		repo:         r,
		tokenStorage: s,
	}
}

func (uc *SoftDeleteProfileUseCase) Execute(input SoftDeleteProfileInput) error {
	if input.UserID == "" {
		return ErrUserIdNotFound
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

	if isPasswordValid, err := security.IsPasswordValid(input.Password, userEntity.PasswordHash); err != nil || !isPasswordValid {
		return ErrInvalidPassword
	}

	userEntity.Deactivate()

	if err := uc.repo.Save(userEntity); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	if err := uc.tokenStorage.RevokeAllUserSessions(userEntity.ID); err != nil {
		log.Printf("[SECURITY WARN] Failed to revoke sessions for user %s after profile deactivation: %v", input.UserID, err)
		return nil
	}

	return nil
}
