package usecase

import (
	"authorization/internal/domain"
	"authorization/internal/domain/security"
	"errors"
	"fmt"
)

var (
	ErrInvalidCredentials      = errors.New("invalid user or password")
	ErrTokenPairGenerationFail = errors.New("failed to generate access and refresh tokens")
	ErrTokenSaveFail           = errors.New("failed to save access and refresh tokens")
	ErrEmailIsNotVerified      = errors.New("Email is not verified")
)

type TokenGenerator interface {
	GenerateTokens(id string) (domain.TokenPair, error)
}

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
	user, err := uc.repo.GetUser(input.Email)
	if err != nil {
		return domain.TokenPair{}, ErrInvalidCredentials
	}

	if isValid, err := security.IsPasswordValid(input.Password, user.PasswordHash); err != nil || !isValid {
		return domain.TokenPair{}, ErrInvalidCredentials
	}

	if !user.IsVerified {
		return domain.TokenPair{}, ErrEmailIsNotVerified
	}

	tokens, err := uc.tokenGenerator.GenerateTokens(user.ID)
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("%w: %v", ErrTokenPairGenerationFail, err)
	}

	if err := uc.tokenStorage.SaveTokenPair(user.ID, tokens); err != nil {
		return domain.TokenPair{}, fmt.Errorf("%w: %v", ErrTokenSaveFail, err)
	}

	return tokens, nil
}
