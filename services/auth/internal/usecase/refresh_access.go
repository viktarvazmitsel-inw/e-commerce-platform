package usecase

import (
	"authorization/internal/domain"
	"errors"
	"fmt"
)

var ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")

type RefreshTokenInput struct {
	Token string
}

type RefreshAccessUseCase struct {
	tokenStorage   TokenRepository
	tokenGenerator TokenGenerator
}

func NewRefreshAccessUseCase(s TokenRepository, g TokenGenerator) *RefreshAccessUseCase {
	return &RefreshAccessUseCase{
		tokenStorage:   s,
		tokenGenerator: g,
	}
}

func (uc *RefreshAccessUseCase) Execute(input RefreshTokenInput) (domain.TokenPair, error) {
	if input.Token == "" {
		return domain.TokenPair{}, ErrInvalidRefreshToken
	}

	userData, err := uc.tokenStorage.GetUserInfoByRefreshToken(input.Token)
	if err != nil {
		return domain.TokenPair{}, ErrInvalidRefreshToken
	}

	tokens, err := uc.tokenGenerator.GenerateTokens(userData.UserID, userData.Role)
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("%w: %v", ErrTokenGenerationFailed, err)
	}

	if err := uc.tokenStorage.UnsetAndSaveTokenPair(input.Token, userData.UserID, tokens); err != nil {
		return domain.TokenPair{}, fmt.Errorf("%w: %v", ErrTokenPairSaveFail, err)
	}

	return tokens, nil
}
