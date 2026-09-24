package usecase

import (
	"authorization/internal/domain"
	"context"
	"fmt"
)

type LoginInput struct {
	Email    string
	Password string
}

type LoginUserUseCase struct {
	repo           UserRepository
	tokenStorage   SessionRepository
	tokenGenerator TokenGenerator
	passwordHasher PasswordHasher
}

func NewLoginUserUseCase(
	r UserRepository,
	s SessionRepository,
	g TokenGenerator,
	h PasswordHasher,
) *LoginUserUseCase {
	return &LoginUserUseCase{
		repo:           r,
		tokenStorage:   s,
		tokenGenerator: g,
		passwordHasher: h,
	}
}

func (uc *LoginUserUseCase) Execute(ctx context.Context, input LoginInput) (domain.TokenPair, error) {
	userEntity, err := uc.repo.GetUserByEmail(ctx, input.Email)
	if err != nil {
		return domain.TokenPair{}, ErrInvalidCredentials
	}

	if !uc.passwordHasher.Compare(input.Password, userEntity.PasswordHash) {
		return domain.TokenPair{}, ErrInvalidCredentials
	}

	if err := userEntity.EnsureActive(); err != nil {
		return domain.TokenPair{}, err
	}

	if !userEntity.IsVerified {
		return domain.TokenPair{}, ErrUserEmailIsNotVerified
	}

	tokens, err := uc.tokenGenerator.GenerateTokens(ctx, userEntity.ID, userEntity.Role)
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("%w: %v", ErrTokenPairGenerationFail, err)
	}

	if err := uc.tokenStorage.SaveTokenPair(ctx, userEntity.ID, tokens); err != nil {
		return domain.TokenPair{}, fmt.Errorf("%w: %v", ErrTokenPairSaveFail, err)
	}

	return tokens, nil
}
