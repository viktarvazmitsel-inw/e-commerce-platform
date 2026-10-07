package usecase

import (
	"authorization/internal/domain/security"
	"context"
	"errors"
	"fmt"
	"strings"
)

type ConfirmEmailUpdateInput struct {
	Token string
}

type ConfirmEmailUpdateResult struct {
	UserID         string
	SessionRevoked bool
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

func (uc *ConfirmEmailUpdateUseCase) Execute(ctx context.Context, input ConfirmEmailUpdateInput) (ConfirmEmailUpdateResult, error) {
	trimmedToken := strings.TrimSpace(input.Token)

	if trimmedToken == "" {
		return ConfirmEmailUpdateResult{}, ErrInvalidUpdateEmailToken
	}

	hashedToken := security.HashToken(trimmedToken)
	tokenData, err := uc.emailTokenStorage.LoadEmailUpdateToken(ctx, hashedToken)
	if err != nil {
		return ConfirmEmailUpdateResult{}, ErrInvalidUpdateEmailToken
	}

	userEntity, err := uc.repo.GetUserByID(ctx, tokenData.UserID)
	if err != nil {
		if errors.Is(err, ErrUserIdNotFound) {
			return ConfirmEmailUpdateResult{}, ErrUserIdNotFound
		}
		return ConfirmEmailUpdateResult{}, fmt.Errorf("failed to fetch user from database: %w", err)
	}

	if err := userEntity.EnsureActive(); err != nil {
		return ConfirmEmailUpdateResult{}, err
	}

	if err := userEntity.UpdateUserEmail(tokenData.NewEmail); err != nil {
		return ConfirmEmailUpdateResult{}, err
	}

	userEntity.VerifyEmail()

	if err := uc.repo.Save(ctx, userEntity); err != nil {
		if errors.Is(err, ErrEmailAlreadyTaken) {
			return ConfirmEmailUpdateResult{}, ErrEmailAlreadyTaken
		}
		return ConfirmEmailUpdateResult{}, fmt.Errorf("unable to save user: %w", err)
	}

	if err := uc.emailTokenStorage.DeleteUserDataUpdateToken(ctx, hashedToken); err != nil {
		uc.logger.Warn(ctx, "unable to delete update email token", "userID", userEntity.ID, "error", err)
	}

	if err := uc.sessionTokenStorage.RevokeAllUserSessions(ctx, tokenData.UserID); err != nil {
		uc.logger.Warn(ctx, "failed to revoke user session", "userID", userEntity.ID, "error", err)
		return ConfirmEmailUpdateResult{UserID: userEntity.ID, SessionRevoked: false}, nil
	}

	return ConfirmEmailUpdateResult{UserID: userEntity.ID, SessionRevoked: true}, nil
}
