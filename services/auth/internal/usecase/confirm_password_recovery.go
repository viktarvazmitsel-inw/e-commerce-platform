package usecase

import (
	"authorization/internal/domain/security"
	"context"
	"errors"
	"fmt"
	"strings"
)

type ConfirmPasswordRecoveryInput struct {
	Token       string
	NewPassword string
}

type ConfirmPasswordRecoveryResult struct {
	UserID         string
	SessionRevoked bool
}

type ConfirmPasswordRecoveryUseCase struct {
	repo           UserRepository
	tokenStorage   UserDataUpdateTokenRepository
	sessionStorage SessionRepository
	passwordHasher PasswordHasher
	logger         Logger
}

func NewConfirmPasswordRecoveryUseCase(
	r UserRepository,
	ts UserDataUpdateTokenRepository,
	h PasswordHasher,
	ss SessionRepository,
	l Logger,
) *ConfirmPasswordRecoveryUseCase {
	return &ConfirmPasswordRecoveryUseCase{
		repo:           r,
		tokenStorage:   ts,
		passwordHasher: h,
		sessionStorage: ss,
		logger:         l,
	}
}

func (uc *ConfirmPasswordRecoveryUseCase) Execute(ctx context.Context, input ConfirmPasswordRecoveryInput) (ConfirmPasswordRecoveryResult, error) {
	trimmedToken := strings.TrimSpace(input.Token)
	if trimmedToken == "" {
		return ConfirmPasswordRecoveryResult{}, ErrInvalidPasswordRecoveryToken
	}

	if !security.IsPasswordStrong(input.NewPassword) {
		return ConfirmPasswordRecoveryResult{}, ErrPasswordIsNotStrongEnough
	}

	hashedResetToken := security.HashToken(trimmedToken)

	userID, err := uc.tokenStorage.LoadPasswordResetToken(ctx, hashedResetToken)
	if err != nil {
		return ConfirmPasswordRecoveryResult{}, ErrInvalidPasswordRecoveryToken
	}

	userEntity, err := uc.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserIdNotFound) {
			return ConfirmPasswordRecoveryResult{}, ErrUserIdNotFound
		}

		return ConfirmPasswordRecoveryResult{}, fmt.Errorf("unable to load user from database: %w", err)
	}

	if err := userEntity.EnsureActive(); err != nil {
		return ConfirmPasswordRecoveryResult{}, err
	}

	if uc.passwordHasher.Compare(input.NewPassword, userEntity.PasswordHash) {
		return ConfirmPasswordRecoveryResult{}, ErrSameNewPassword
	}

	newPasswordHash, err := uc.passwordHasher.Hash(input.NewPassword)
	if err != nil {
		return ConfirmPasswordRecoveryResult{}, ErrPasswordHashFail
	}

	if err := userEntity.UpdateUserPassword(newPasswordHash); err != nil {
		return ConfirmPasswordRecoveryResult{}, err
	}

	if err := uc.repo.Save(ctx, userEntity); err != nil {
		return ConfirmPasswordRecoveryResult{}, fmt.Errorf("unable to save user: %w", err)
	}

	if err := uc.tokenStorage.DeleteUserDataUpdateToken(ctx, hashedResetToken); err != nil {
		uc.logger.Warn(ctx, "failed to delete password recovery token", "userID", userEntity.ID, "error", err)
	}

	if err := uc.sessionStorage.RevokeAllUserSessions(ctx, userEntity.ID); err != nil {
		uc.logger.Warn(ctx, "failed to revoke user sessions", "userID", userEntity.ID, "error", err)
		return ConfirmPasswordRecoveryResult{UserID: userEntity.ID, SessionRevoked: false}, nil
	}

	return ConfirmPasswordRecoveryResult{UserID: userEntity.ID, SessionRevoked: true}, nil
}
