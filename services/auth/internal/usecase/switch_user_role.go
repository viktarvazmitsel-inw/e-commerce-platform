package usecase

import (
	"authorization/internal/domain"
	"errors"
	"fmt"
)

type SwitchUserRoleInput struct {
	AuthorID string
	TargetID string
	NewRole  domain.Role
}

type SwitchUserRoleUseCase struct {
	repo UserRepository
}

func NewSwitchUserRoleUseCase(r UserRepository) *SwitchUserRoleUseCase {
	return &SwitchUserRoleUseCase{
		repo: r,
	}
}

func (uc *SwitchUserRoleUseCase) Execute(input SwitchUserRoleInput) error {
	if input.AuthorID == "" {
		return ErrUnauthorizedAction
	}
	if input.TargetID == "" {
		return ErrUserIdNotFound
	}

	authorUser, err := uc.repo.GetUserById(input.AuthorID)
	if err != nil {
		if errors.Is(err, ErrUserIdNotFound) {
			return ErrUserIdNotFound
		}

		return fmt.Errorf("failed to fetch user from database: %w", err)
	}

	if err := authorUser.RequirePermission(domain.PermissionChangeRole); err != nil {
		return err
	}

	targetUser, err := uc.repo.GetUserById(input.TargetID)
	if err != nil {
		if errors.Is(err, ErrUserIdNotFound) {
			return ErrUserIdNotFound
		}

		return fmt.Errorf("failed to fetch user from database: %w", err)
	}

	if err := targetUser.EnsureActive(); err != nil {
		return err
	}

	if err := targetUser.SetRole(input.NewRole); err != nil {
		return err
	}

	if err := uc.repo.Save(targetUser); err != nil {
		return fmt.Errorf("unable to save user: %w", err)
	}

	return nil
}
