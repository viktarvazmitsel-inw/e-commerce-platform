package usecase

import (
	"authorization/internal/domain"
	"authorization/internal/domain/security"
	"errors"
	"fmt"
)

var (
	ErrInvalidCredentials = errors.New("invalid user or password")
	ErrEmailIsNotVerified = errors.New("Email is not verified")
)

type LoginInput struct {
	Email    string
	Password string
}

type LoginUserUseCase struct {
	repo           UserRepository
	tokenStorage   TokenRepository
	tokenGenerator TokenGenerator
}

func NewLoginUserUseCase(r UserRepository, s TokenRepository, g TokenGenerator) *LoginUserUseCase {
	return &LoginUserUseCase{
		repo:           r,
		tokenStorage:   s,
		tokenGenerator: g,
	}
}

func (uc *LoginUserUseCase) Execute(input LoginInput) (domain.TokenPair, error) {
	userEntity, err := uc.repo.GetUserByEmail(input.Email)
	if err != nil {
		return domain.TokenPair{}, ErrInvalidCredentials
	}

	if err := userEntity.EnsureActive(); err != nil {
		return domain.TokenPair{}, err
	}

	if isValid, err := security.IsPasswordValid(input.Password, userEntity.PasswordHash); err != nil || !isValid {
		return domain.TokenPair{}, ErrInvalidCredentials
	}

	if !userEntity.IsVerified {
		return domain.TokenPair{}, ErrEmailIsNotVerified
	}

	tokens, err := uc.tokenGenerator.GenerateTokens(userEntity.ID, userEntity.Role)
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("%w: %v", ErrTokenPairGenerationFail, err)
	}

	if err := uc.tokenStorage.SaveTokenPair(userEntity.ID, tokens); err != nil {
		return domain.TokenPair{}, fmt.Errorf("%w: %v", ErrTokenPairSaveFail, err)
	}

	return tokens, nil
}
