package usecase

import "fmt"

type LogoutUserInput struct {
	UserID string
}

type LogoutUserUseCase struct {
	tokenStorage TokenRepository
}

func NewLogoutUserUseCase(s TokenRepository) *LogoutUserUseCase {
	return &LogoutUserUseCase{
		tokenStorage: s,
	}
}

func (uc *LogoutUserUseCase) Execute(input LogoutUserInput) error {
	if input.UserID == "" {
		return ErrUserIdNotFound
	}

	if err := uc.tokenStorage.RevokeAllUserSessions(input.UserID); err != nil {
		return fmt.Errorf("failed to revoke user session: %w", err)
	}

	return nil
}
