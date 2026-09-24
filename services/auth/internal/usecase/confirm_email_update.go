package usecase

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidUpdateEmailToken = errors.New("invalid update token")
)

type ConfirmEmailUpdateInput struct {
	Token string
}

type ConfirmEmailUpdateUseCase struct {
	repo         UserRepository
	tokenStorage TokenRepository
}

func NewConfirmEmailUpdateUseCase(r UserRepository, s TokenRepository) *ConfirmEmailUpdateUseCase {
	return &ConfirmEmailUpdateUseCase{
		repo:         r,
		tokenStorage: s,
	}
}

func (uc *ConfirmEmailUpdateUseCase) Execute(input ConfirmEmailUpdateInput) error {
	if input.Token == "" {
		return ErrInvalidUpdateEmailToken
	}

	tokenData, err := uc.tokenStorage.LoadEmailUpdateToken(input.Token)
	if err != nil {
		return ErrInvalidUpdateEmailToken
	}

	userEntity, err := uc.repo.GetUserById(tokenData.UserID)
	if err != nil {
		if errors.Is(err, ErrUserIdNotFound) {
			return ErrUserIdNotFound
		}
		return fmt.Errorf("failed to fetch user from database: %w", err)
	}

	if err := userEntity.UpdateUserEmail(tokenData.NewEmail); err != nil {
		return err
	}

	if err := uc.repo.Save(userEntity); err != nil {
		return fmt.Errorf("unable to save user: %w", err)
	}

	if err := uc.tokenStorage.DeleteEmailUpdateToken(input.Token); err != nil {
		return fmt.Errorf("failed to delete email update token: %w", err)
	}

	if err := uc.tokenStorage.RevokeAllUserSessions(tokenData.UserID); err != nil {
		return fmt.Errorf("failed to revoke user session: %w", err)
	}

	return nil
}
