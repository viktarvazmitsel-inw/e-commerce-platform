package usecase

import (
	"authorization/internal/domain"
	"context"
	"errors"
	"fmt"
	"strings"
)

type SoftDeleteProfileInput struct {
	AuthorID string
	TargetID string
	Password string
}

type SoftDeleteProfileResult struct {
	TargetID       string
	SessionRevoked bool
}

type SoftDeleteProfileUseCase struct {
	repo           UserRepository
	tokenStorage   SessionRepository
	passwordHasher PasswordHasher
	logger         Logger
}

func NewSoftDeleteProfileUseCase(
	r UserRepository,
	s SessionRepository,
	h PasswordHasher,
	l Logger,
) *SoftDeleteProfileUseCase {
	return &SoftDeleteProfileUseCase{
		repo:           r,
		tokenStorage:   s,
		passwordHasher: h,
		logger:         l,
	}
}

func (uc *SoftDeleteProfileUseCase) Execute(ctx context.Context, input SoftDeleteProfileInput) (SoftDeleteProfileResult, error) {
	trimmedAuthorID := strings.TrimSpace(input.AuthorID)
	if trimmedAuthorID == "" {
		return SoftDeleteProfileResult{}, ErrUserIdNotFound
	}

	trimmedTargetID := strings.TrimSpace(input.TargetID)
	if trimmedTargetID == "" {
		return SoftDeleteProfileResult{}, ErrTargetUserNotFound
	}

	authorEntity, err := uc.repo.GetUserByID(ctx, trimmedAuthorID)
	if err != nil {
		if errors.Is(err, ErrUserIdNotFound) {
			return SoftDeleteProfileResult{}, ErrUserIdNotFound
		}

		return SoftDeleteProfileResult{}, fmt.Errorf("failed to fetch user from database: %w", err)
	}

	if err := authorEntity.EnsureActive(); err != nil {
		return SoftDeleteProfileResult{}, err
	}

	if !uc.passwordHasher.Compare(input.Password, authorEntity.PasswordHash) {
		return SoftDeleteProfileResult{}, ErrInvalidPassword
	}

	isSelf := trimmedAuthorID == trimmedTargetID
	isAdmin := authorEntity.Can(domain.PermissionDeleteUserProfile)
	if !isSelf && !isAdmin {
		return SoftDeleteProfileResult{}, ErrUnauthorizedAction
	}

	var targetEntity *domain.User
	if isSelf {
		targetEntity = authorEntity
	} else {
		targetEntity, err = uc.repo.GetUserByID(ctx, trimmedTargetID)
		if err != nil {
			if errors.Is(err, ErrUserIdNotFound) {
				return SoftDeleteProfileResult{}, ErrTargetUserNotFound
			}
			return SoftDeleteProfileResult{}, fmt.Errorf("failed to fetch user from database: %w", err)
		}

		if err := targetEntity.EnsureActive(); err != nil {
			return SoftDeleteProfileResult{}, err
		}
	}

	targetEntity.Deactivate()
	if err := uc.repo.Save(ctx, targetEntity); err != nil {
		return SoftDeleteProfileResult{}, fmt.Errorf("unable to save user: %w", err)
	}
	if err := uc.tokenStorage.RevokeAllUserSessions(ctx, targetEntity.ID); err != nil {
		uc.logger.Warn(ctx, "failed to revoke user session", "userID", targetEntity.ID, "error", err)
		return SoftDeleteProfileResult{TargetID: targetEntity.ID, SessionRevoked: false}, nil
	}
	return SoftDeleteProfileResult{TargetID: targetEntity.ID, SessionRevoked: true}, nil

}
