package usecase

import (
	"authorization/internal/domain"
	"context"
	"fmt"
)

type RefreshTokenInput struct {
	Token string
}

type RefreshAccessUseCase struct {
	repo           UserRepository
	tokenStorage   SessionRepository
	tokenGenerator TokenGenerator
}

func NewRefreshAccessUseCase(r UserRepository, s SessionRepository, g TokenGenerator) *RefreshAccessUseCase {
	return &RefreshAccessUseCase{
		repo:           r,
		tokenStorage:   s,
		tokenGenerator: g,
	}
}

func (uc *RefreshAccessUseCase) Execute(ctx context.Context, input RefreshTokenInput) (domain.TokenPair, error) {
	if input.Token == "" {
		return domain.TokenPair{}, ErrInvalidRefreshToken
	}

	userData, err := uc.tokenStorage.GetUserInfoByRefreshToken(ctx, input.Token)
	if err != nil {
		return domain.TokenPair{}, ErrInvalidRefreshToken
	}

	userEntity, err := uc.repo.GetUserByID(ctx, userData.UserID)
	if err != nil {
		return domain.TokenPair{}, ErrUserIdNotFound
	}

	if err := userEntity.EnsureActive(); err != nil {
		return domain.TokenPair{}, err
	}

	tokens, err := uc.tokenGenerator.GenerateTokens(ctx, userEntity.ID, userEntity.Role)
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("%w: %v", ErrTokenGenerationFailed, err)
	}

	if err := uc.tokenStorage.UnsetAndSaveTokenPair(ctx, input.Token, userData.UserID, tokens); err != nil {
		return domain.TokenPair{}, fmt.Errorf("%w: %v", ErrTokenPairSaveFail, err)
	}

	return tokens, nil
}
