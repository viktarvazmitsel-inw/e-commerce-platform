package usecase

import (
	"errors"
	"fmt"
)

var ErrUserEmailIsNotVerified = errors.New("user email is not verified yet")

type UpdateUserProfileInput struct {
	ID      string
	Name    string
	Surname string
	Phone   string
}

type UpdateUserProfileUseCase struct {
	repo UserRepository
}

func NewUpdateUserProfileUseCase(r UserRepository) *UpdateUserProfileUseCase {
	return &UpdateUserProfileUseCase{
		repo: r,
	}
}

func (uc *UpdateUserProfileUseCase) Execute(input UpdateUserProfileInput) error {
	if input.ID == "" {
		return ErrUserIdNotFound
	}

	userEntity, err := uc.repo.GetUserById(input.ID)
	if err != nil {
		if errors.Is(err, ErrUserIdNotFound) {
			return ErrUserIdNotFound
		}
		return fmt.Errorf("failed to fetch user from database: %w", err)
	}

	if err := userEntity.EnsureActive(); err != nil {
		return err
	}

	if !userEntity.IsVerified {
		return ErrUserEmailIsNotVerified
	}

	if err := userEntity.UpdateUser(input.Name, input.Surname, input.Phone); err != nil {
		return err
	}

	if err := uc.repo.Save(userEntity); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	return nil
}
