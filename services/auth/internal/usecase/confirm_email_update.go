package usecase

import (
	"context"
	"errors"
	"fmt"
)

type ConfirmEmailUpdateInput struct {
	Token string
}

type ConfirmEmailUpdateUseCase struct {
	repo                UserRepository
	emailTokenStorage   EmailUpdateTokenRepository
	sessionTokenStorage SessionRepository
}

func NewConfirmEmailUpdateUseCase(
	r UserRepository,
	es EmailUpdateTokenRepository,
	ss SessionRepository,
) *ConfirmEmailUpdateUseCase {
	return &ConfirmEmailUpdateUseCase{
		repo:                r,
		emailTokenStorage:   es,
		sessionTokenStorage: ss,
	}
}

func (uc *ConfirmEmailUpdateUseCase) Execute(ctx context.Context, input ConfirmEmailUpdateInput) error {
	if input.Token == "" {
		return ErrInvalidUpdateEmailToken
	}

	tokenData, err := uc.emailTokenStorage.LoadEmailUpdateToken(ctx, input.Token)
	if err != nil {
		return ErrInvalidUpdateEmailToken
	}

	userEntity, err := uc.repo.GetUserByID(ctx, tokenData.UserID)
	if err != nil {
		if errors.Is(err, ErrUserIdNotFound) {
			return ErrUserIdNotFound
		}
		return fmt.Errorf("failed to fetch user from database: %w", err)
	}

	if err := userEntity.EnsureActive(); err != nil {
		return err
	}

	if err := userEntity.UpdateUserEmail(tokenData.NewEmail); err != nil {
		if errors.Is(err, ErrEmailAlreadyTaken) {
			return ErrEmailAlreadyTaken
		}

		return fmt.Errorf("unable to save user: %w", err)
	}

	if err := uc.repo.Save(ctx, userEntity); err != nil {
		return fmt.Errorf("unable to save user: %w", err)
	}

	if err := uc.emailTokenStorage.DeleteEmailUpdateToken(ctx, input.Token); err != nil {
		return fmt.Errorf("failed to delete email update token: %w", err)
	}

	if err := uc.sessionTokenStorage.RevokeAllUserSessions(ctx, tokenData.UserID); err != nil {
		return fmt.Errorf("failed to revoke user session: %w", err)
	}

	return nil
}
