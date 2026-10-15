package usecase

import (
	"authorization/internal/domain"
	"context"
	"errors"
	"fmt"
	"strings"
)

type UserProfile struct {
	ID         string      `json:"id,omitempty"`
	Email      string      `json:"email,omitempty"`
	Name       string      `json:"name"`
	Surname    string      `json:"surname"`
	Phone      string      `json:"phone,omitempty"`
	Role       domain.Role `json:"role"`
	IsActive   *bool       `json:"is_active"`
	IsVerified *bool       `json:"is_verified"`
}

type GetUserInput struct {
	AuthorID string
	TargetID string
}

type GetUserProfileUseCase struct {
	repo UserRepository
}

func NewGetUserUseCase(r UserRepository) *GetUserProfileUseCase {
	return &GetUserProfileUseCase{
		repo: r,
	}
}

func toProfile(u *domain.User, canViewFull bool) *UserProfile {
	if u == nil {
		return nil
	}

	if canViewFull {
		return &UserProfile{
			ID:         u.ID,
			Email:      u.Email,
			Name:       u.Name,
			Surname:    u.Surname,
			Phone:      u.Phone,
			Role:       u.Role,
			IsActive:   &u.IsActive,
			IsVerified: &u.IsVerified,
		}
	}

	return &UserProfile{
		Name:    u.Name,
		Surname: u.Surname,
		Role:    u.Role,
	}

}

func (uc *GetUserProfileUseCase) Execute(ctx context.Context, input GetUserInput) (*UserProfile, error) {
	trimmedAuthorID := strings.TrimSpace(input.AuthorID)
	if trimmedAuthorID == "" {
		return nil, ErrUserIdNotFound
	}

	trimmedTargetID := strings.TrimSpace(input.TargetID)
	if trimmedTargetID == "" {
		return nil, ErrTargetUserNotFound
	}

	authorEntity, err := uc.repo.GetUserByID(ctx, trimmedAuthorID)
	if err != nil {
		if errors.Is(err, ErrUserIdNotFound) {
			return nil, ErrUserIdNotFound
		}

		return nil, fmt.Errorf("failed to fetch user from database: %w", err)
	}

	if err := authorEntity.EnsureActive(); err != nil {
		return nil, err
	}
	var isSelf = false
	var targetEntity *domain.User
	if trimmedAuthorID == trimmedTargetID {
		targetEntity = authorEntity
		isSelf = true
	} else {
		var err error
		targetEntity, err = uc.repo.GetUserByID(ctx, trimmedTargetID)
		if err != nil {
			if errors.Is(err, ErrUserIdNotFound) {
				return nil, ErrTargetUserNotFound
			}

			return nil, fmt.Errorf("failed to fetch user from database: %w", err)
		}
	}

	isAdmin := authorEntity.Can(domain.PermissionGetFullUserProfile)

	if err := targetEntity.EnsureActive(); err != nil && !isAdmin {
		return nil, ErrTargetUserNotFound
	}

	canViewFull := isAdmin || isSelf

	userProfile := toProfile(targetEntity, canViewFull)
	if userProfile == nil {
		return nil, ErrTargetUserNotFound
	}

	return userProfile, nil
}
