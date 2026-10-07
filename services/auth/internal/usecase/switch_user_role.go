package usecase

import (
	"authorization/internal/domain"
	"context"
	"errors"
	"fmt"
)

type SwitchUserRoleInput struct {
	AuthorID string
	TargetID string
	NewRole  domain.Role
}

type SwitchUserRoleResult struct {
	TargetID       string
	SessionRevoked bool
}

type SwitchUserRoleUseCase struct {
	repo         UserRepository
	tokenStorage SessionRepository
	logger       Logger
}

func NewSwitchUserRoleUseCase(r UserRepository, s SessionRepository, l Logger) *SwitchUserRoleUseCase {
	return &SwitchUserRoleUseCase{
		repo:         r,
		tokenStorage: s,
		logger:       l,
	}
}

func (uc *SwitchUserRoleUseCase) Execute(ctx context.Context, input SwitchUserRoleInput) (SwitchUserRoleResult, error) {
	if input.AuthorID == "" {
		return SwitchUserRoleResult{}, ErrUnauthorizedAction
	}
	if input.TargetID == "" {
		return SwitchUserRoleResult{}, ErrUserIdNotFound
	}

	authorUser, err := uc.repo.GetUserByID(ctx, input.AuthorID)
	if err != nil {
		if errors.Is(err, ErrUserIdNotFound) {
			return SwitchUserRoleResult{}, ErrUserIdNotFound
		}

		return SwitchUserRoleResult{}, fmt.Errorf("failed to fetch user from database: %w", err)
	}

	if err := authorUser.EnsureActive(); err != nil {
		return SwitchUserRoleResult{}, err
	}

	if err := authorUser.RequirePermission(domain.PermissionChangeRole); err != nil {
		return SwitchUserRoleResult{}, err
	}

	targetUser, err := uc.repo.GetUserByID(ctx, input.TargetID)
	if err != nil {
		if errors.Is(err, ErrUserIdNotFound) {
			return SwitchUserRoleResult{}, ErrUserIdNotFound
		}

		return SwitchUserRoleResult{}, fmt.Errorf("failed to fetch user from database: %w", err)
	}

	if err := targetUser.EnsureActive(); err != nil {
		return SwitchUserRoleResult{}, err
	}

	if err := targetUser.SetRole(input.NewRole); err != nil {
		return SwitchUserRoleResult{}, err
	}

	if err := uc.repo.Save(ctx, targetUser); err != nil {
		return SwitchUserRoleResult{}, fmt.Errorf("unable to save user: %w", err)
	}

	if err := uc.tokenStorage.RevokeAllUserSessions(ctx, input.TargetID); err != nil {
		uc.logger.Warn(ctx, "unable to revoke user access token", "authorID", authorUser.ID, "targetID", targetUser.ID, "error", err)
		return SwitchUserRoleResult{TargetID: targetUser.ID, SessionRevoked: false}, nil
	}

	return SwitchUserRoleResult{TargetID: targetUser.ID, SessionRevoked: true}, nil
}
