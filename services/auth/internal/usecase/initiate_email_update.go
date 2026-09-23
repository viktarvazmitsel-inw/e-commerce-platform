package usecase

import (
	"authorization/internal/domain"
	"authorization/internal/domain/security"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrSameEmail = errors.New("new email must be different from current")

type EmailUpdateInput struct {
	ID       string
	Password string
	NewEmail string
}

type InitiateEmailUpdateUseCase struct {
	repo         UserRepository
	mail         EmailSender
	tokenStorage TokenRepository
}

func NewInitiateEmailUpdateUseCase(r UserRepository, m EmailSender, s TokenRepository) *InitiateEmailUpdateUseCase {
	return &InitiateEmailUpdateUseCase{
		repo:         r,
		mail:         m,
		tokenStorage: s,
	}
}

func (uc *InitiateEmailUpdateUseCase) Execute(input EmailUpdateInput) error {
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

	if isPasswordValid, err := security.IsPasswordValid(input.Password, userEntity.PasswordHash); err != nil || !isPasswordValid {
		return ErrInvalidPassword
	}

	trimmedNewEmail := strings.TrimSpace(input.NewEmail)

	if trimmedNewEmail == userEntity.Email {
		return ErrSameEmail
	}

	if !domain.IsEmailValid(trimmedNewEmail) {
		return domain.ErrInvalidEmail
	}

	if uc.repo.IsEmailTaken(trimmedNewEmail) {
		return ErrEmailAlreadyTaken
	}

	token, err := security.GenerateEmailVerificationToken(userEntity.ID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTokenGenerationFailed, err)
	}

	if err := uc.tokenStorage.SaveEmailUpdateToken(userEntity.ID, trimmedNewEmail, token, time.Hour); err != nil {
		return fmt.Errorf("%w: %v", ErrVerificationTokenSaveFailed, err)
	}

	if err := uc.mail.SendVerificationEmail(trimmedNewEmail, token); err != nil {
		return fmt.Errorf("%w: %v", ErrVerificationEmailSendFailed, err)
	}

	return nil

}
