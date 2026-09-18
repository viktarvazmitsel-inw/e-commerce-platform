package usecase

import (
	"authorization/internal/domain"
	"authorization/internal/domain/security"
	"errors"
	"fmt"
	"time"
	"unicode"
)

var (
	ErrPasswordIsNotStrongEnough   = errors.New("password must contain upper and lowercase letters, numbers and symbols")
	ErrPasswordHashFailed          = errors.New("password hash failed.")
	ErrEmailAlreadyTaken           = errors.New("email already taken")
	ErrTokenGenerationFailed       = errors.New("registration failed due to token generation fail.")
	ErrVerificationTokenSaveFailed = errors.New("failed to save verification token.")
	ErrVerificationEmailSendFailed = errors.New("failed to send verification email.")
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

func IsPasswordStrong(p string) bool {
	if len(p) < 8 {
		return false
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range p {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasDigit && hasSpecial
}

func NewRegisterUserUseCase(r UserRepository, m EmailSender, s TokenRepository) *RegisterUserUseCase {
	return &RegisterUserUseCase{
		repo:         r,
		mail:         m,
		tokenStorage: s,
	}
}

func (uc *RegisterUserUseCase) Execute(input RegisterInput) error {
	if !IsPasswordStrong(input.Password) {
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
		return err
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
