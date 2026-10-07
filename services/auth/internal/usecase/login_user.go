package usecase

import (
	"authorization/internal/domain"
	"context"
	"errors"
	"fmt"
	"strings"
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
	trimmedEmail := strings.TrimSpace(input.Email)

	userEntity, err := uc.repo.GetUserByEmail(ctx, trimmedEmail)
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
		return domain.TokenPair{}, ErrTokenPairGenerationFail
	}

	userInfo := &UserInfoTokenData{
		UserID: userEntity.ID,
		Role:   userEntity.Role,
	}

	if err := uc.tokenStorage.SaveTokenPair(ctx, userInfo, tokens); err != nil {
		if errors.Is(err, ErrTokenPairSaveFail) {
			return domain.TokenPair{}, ErrTokenPairSaveFail
		}

		return domain.TokenPair{}, fmt.Errorf("failed to save token pair to token storage: %w", err)
	}

	return tokens, nil
}
