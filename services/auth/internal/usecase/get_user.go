package usecase

import (
	"authorization/internal/domain"
	"errors"
	"fmt"
)

var ErrUserIdNotFound = errors.New("user with such id not found")

type UserProfile struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Surname string `json:"surname"`
	Phone   string `json:"phone"`
	Role    int    `json:"role"`
}

type GetUserInput struct {
	ID string
}

type GetUserUseCase struct {
	repo UserRepository
}

func NewGetUserUseCase(r UserRepository) *GetUserUseCase {
	return &GetUserUseCase{
		repo: r,
	}
}

func toProfile(u *domain.User) *UserProfile {
	if u == nil {
		return nil
	}
	return &UserProfile{
		ID:      u.ID,
		Email:   u.Email,
		Name:    u.Name,
		Surname: u.Surname,
		Phone:   u.Phone,
		Role:    u.Role,
	}
}

func (uc *GetUserUseCase) Execute(input GetUserInput) (*UserProfile, error) {
	if input.ID == "" {
		return nil, ErrUserIdNotFound
	}

	userData, err := uc.repo.GetUserById(input.ID)
	if err != nil {
		if errors.Is(err, ErrUserIdNotFound) {
			return nil, ErrUserIdNotFound
		}

		return nil, fmt.Errorf("failed to fetch user from database: %w", err)
	}

	userProfile := toProfile(userData)
	if userProfile == nil {
		return nil, ErrUserIdNotFound
	}

	return userProfile, nil
}
