package usecase

import (
	"context"
	"fmt"
)

type LogoutUserInput struct {
	UserID string
	Token  string
}

type LogoutUserUseCase struct {
	tokenStorage SessionRepository
}

func NewLogoutUserUseCase(s SessionRepository) *LogoutUserUseCase {
	return &LogoutUserUseCase{
		tokenStorage: s,
	}
}

func (uc *LogoutUserUseCase) Execute(ctx context.Context, input LogoutUserInput) error {
	if input.UserID == "" {
		return ErrUserIdNotFound
	}

	if input.Token == "" {
		return ErrInvalidRefreshToken
	}

	if err := uc.tokenStorage.RevokeCurrentUserSession(ctx, input.UserID, input.Token); err != nil {
		return fmt.Errorf("failed to revoke user session: %w", err)
	}

	return nil
}
