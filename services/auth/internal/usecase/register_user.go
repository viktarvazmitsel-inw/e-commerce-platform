package usecase

import (
	"authorization/internal/domain"
	"authorization/internal/domain/security"
	"fmt"
	"time"
)

type EmailSender interface {
	SendVerificationEmail(email string, token string) error
}

type RegisterInput struct {
	Email    string
	Password string
	Name     string
	Surname  string
	Phone    string
}

type RegisterUserUseCase struct {
	repo         UserRepository
	mail         EmailSender
	tokenStorage TokenRepository
}

func NewRegisterUserUseCase(r UserRepository, m EmailSender, s TokenRepository) *RegisterUserUseCase {
	return &RegisterUserUseCase{
		repo:         r,
		mail:         m,
		tokenStorage: s,
	}
}

func (uc *RegisterUserUseCase) Execute(input RegisterInput) error {
	if !security.IsPasswordStrong(input.Password) {
		return ErrPasswordIsNotStrongEnough
	}

	passwordHash, err := security.HashPassword(input.Password)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPasswordHashFailed, err)
	}

	if uc.repo.IsEmailTaken(input.Email) {
		return ErrEmailAlreadyTaken
	}

	userEntity, err := domain.NewUser(input.Email, passwordHash, input.Name, input.Surname, input.Phone)
	if err != nil {
		return err
	}

	if err := uc.repo.Save(userEntity); err != nil {
		return fmt.Errorf("unable to save user: %w", err)
	}

	token, err := security.GenerateEmailVerificationToken(userEntity.ID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTokenGenerationFailed, err)
	}

	if err := uc.tokenStorage.SaveVerificationToken(userEntity.ID, token, time.Hour); err != nil {
		return fmt.Errorf("%w: %v", ErrVerificationTokenSaveFailed, err)
	}

	if err := uc.mail.SendVerificationEmail(userEntity.Email, token); err != nil {
		return fmt.Errorf("%w: %v", ErrVerificationEmailSendFailed, err)
	}

	return nil
}
