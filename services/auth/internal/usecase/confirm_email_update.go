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
	emailTokenStorage   UserDataUpdateTokenRepository
	sessionTokenStorage SessionRepository
	logger              Logger
}

func NewConfirmEmailUpdateUseCase(
	r UserRepository,
	es UserDataUpdateTokenRepository,
	ss SessionRepository,
	l Logger,
) *ConfirmEmailUpdateUseCase {
	return &ConfirmEmailUpdateUseCase{
		repo:                r,
		emailTokenStorage:   es,
		sessionTokenStorage: ss,
		logger:              l,
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
		return err
	}

	userEntity.VerifyEmail()

	if err := uc.repo.Save(ctx, userEntity); err != nil {
		if errors.Is(err, ErrEmailAlreadyTaken) {
			return ErrEmailAlreadyTaken
		}
		return fmt.Errorf("unable to save user: %w", err)
	}

	if err := uc.emailTokenStorage.DeleteUserDataUpdateToken(ctx, input.Token); err != nil {
		uc.logger.Warn(ctx, "unable to delete update email token", "userID", userEntity.ID, "error", err)
	}

	if err := uc.sessionTokenStorage.RevokeAllUserSessions(ctx, tokenData.UserID); err != nil {
		uc.logger.Warn(ctx, "failed to revoke user session", "userID", userEntity.ID, "error", err)
		return nil
	}

	return nil
}
