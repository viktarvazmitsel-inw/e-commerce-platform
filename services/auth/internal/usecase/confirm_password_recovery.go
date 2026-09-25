package usecase

import (
	"authorization/internal/domain/security"
	"context"
	"errors"
	"fmt"
)

type ConfirmPasswordRecoveryInput struct {
	Token       string
	NewPassword string
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

func (uc *ConfirmPasswordRecoveryUseCase) Execute(ctx context.Context, input ConfirmPasswordRecoveryInput) error {
	if input.Token == "" {
		return ErrInvalidPasswordRecoveryToken
	}

	if !security.IsPasswordStrong(input.NewPassword) {
		return ErrPasswordIsNotStrongEnough
	}

	userData, err := uc.tokenStorage.LoadPasswordResetToken(ctx, input.Token)
	if err != nil {
		return ErrInvalidPasswordRecoveryToken
	}

	userEntity, err := uc.repo.GetUserByID(ctx, userData.UserID)
	if err != nil {
		if errors.Is(err, ErrUserIdNotFound) {
			return ErrUserIdNotFound
		}

		return fmt.Errorf("unable to load user from database: %w", err)
	}

	if err := userEntity.EnsureActive(); err != nil {
		return err
	}

	if uc.passwordHasher.Compare(input.NewPassword, userEntity.PasswordHash) {
		return ErrSameNewPassword
	}

	newPasswordHash, err := uc.passwordHasher.Hash(input.NewPassword)
	if err != nil {
		return ErrPasswordHashFail
	}

	if err := userEntity.UpdateUserPassword(newPasswordHash); err != nil {
		return err
	}

	if err := uc.repo.Save(ctx, userEntity); err != nil {
		return fmt.Errorf("unable to save user: %w", err)
	}

	if err := uc.tokenStorage.DeleteUserDataUpdateToken(ctx, input.Token); err != nil {
		uc.logger.Warn(ctx, "failed to delete password recovery token", "userID", userEntity.ID, "error", err)
		return nil
	}

	if err := uc.sessionStorage.RevokeAllUserSessions(ctx, userEntity.ID); err != nil {
		uc.logger.Warn(ctx, "failed to revoke user sessions", "userID", userEntity.ID, "error", err)
		return nil
	}

	return nil
}
