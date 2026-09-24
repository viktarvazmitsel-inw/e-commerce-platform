package usecase

import (
	"context"
	"errors"
	"fmt"
)

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

func (uc *UpdateUserProfileUseCase) Execute(ctx context.Context, input UpdateUserProfileInput) error {
	if input.ID == "" {
		return ErrUserIdNotFound
	}

	userEntity, err := uc.repo.GetUserByID(ctx, input.ID)
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

	if err := uc.repo.Save(ctx, userEntity); err != nil {
		return fmt.Errorf("unable to save user: %w", err)
	}

	return nil
}
