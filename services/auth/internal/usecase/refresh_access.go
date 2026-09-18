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

	userId, err := uc.tokenStorage.GetUserIDByRefreshToken(input.Token)
	if err != nil {
		return domain.TokenPair{}, ErrInvalidRefreshToken
	}

	tokens, err := uc.tokenGenerator.GenerateTokens(userId)
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("%w: %v", ErrTokenGenerationFailed, err)
	}

	if err := uc.tokenStorage.UnsetAndSaveTokenPair(input.Token, userId, tokens); err != nil {
		return domain.TokenPair{}, fmt.Errorf("%w: %v", ErrTokenPairSaveFail, err)
	}

	return tokens, nil
}
