package usecase

import (
	"authorization/internal/domain"
	"context"
	"errors"
	"fmt"
	"strings"
)

type ListUsersInput struct {
	AuthorID string
	Quantity int
	Page     int
}

type UserLiteProfile struct {
	ID         string      `json:"id"`
	Email      string      `json:"email"`
	Name       string      `json:"name"`
	Surname    string      `json:"surname"`
	Role       domain.Role `json:"role"`
	IsActive   bool        `json:"is_active"`
	IsVerified bool        `json:"is_verified"`
}

type ListUsersOutput struct {
	Users      []UserLiteProfile `json:"users"`
	Page       int               `json:"page"`
	Quantity   int               `json:"quantity"`
	TotalCount int               `json:"total_count"`
}

type ListUsersUseCase struct {
	repo UserRepository
}

func NewListUsersUseCase(r UserRepository) *ListUsersUseCase {
	return &ListUsersUseCase{
		repo: r,
	}
}

func toUserLiteProfile(u *domain.User) *UserLiteProfile {
	return &UserLiteProfile{
		ID:         u.ID,
		Email:      u.Email,
		Name:       u.Name,
		Surname:    u.Surname,
		Role:       u.Role,
		IsActive:   u.IsActive,
		IsVerified: u.IsVerified,
	}
}

func (l *ListUsersUseCase) Execute(ctx context.Context, input ListUsersInput) (*ListUsersOutput, error) {
	trimmedID := strings.TrimSpace(input.AuthorID)
	if trimmedID == "" {
		return nil, ErrUnauthorizedAction
	}

	page := input.Page
	if input.Page <= 0 {
		page = 1
	}

	quantity := input.Quantity
	if input.Quantity <= 0 || input.Quantity > 100 {
		quantity = 20
	}

	author, err := l.repo.GetUserByID(ctx, trimmedID)
	if err != nil {
		if errors.Is(err, ErrUserIdNotFound) {
			return nil, ErrUserIdNotFound
		}

		return nil, fmt.Errorf("failed to fetch user from database: %w", err)
	}

	if err := author.EnsureActive(); err != nil {
		return nil, err
	}

	if err := author.RequirePermission(domain.PermissionGetUserList); err != nil {
		return nil, err
	}

	userListRawData, resultQuantity, err := l.repo.GetUserList(ctx, page, quantity)
	if err != nil {
		return nil, fmt.Errorf("failed to load users from database: %w", err)
	}
	if userListRawData == nil {
		return nil, errors.New("user repository returned empty result")
	}

	profiles := make([]UserLiteProfile, 0, len(userListRawData))

	for _, userEntity := range userListRawData {
		profiles = append(profiles, *toUserLiteProfile(&userEntity))
	}

	return &ListUsersOutput{
		Users:      profiles,
		Page:       page,
		Quantity:   quantity,
		TotalCount: resultQuantity,
	}, nil

}
