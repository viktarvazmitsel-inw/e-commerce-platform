package domain

import (
	"authorization/internal/domain/security"
	"errors"
	"regexp"
	"strings"
)

const (
	RoleClient    = 1
	RoleAnalycist = 2
	RoleAdmin     = 3
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
var phoneRegex = regexp.MustCompile(`^\+?\d{1,4}?[-.\s]?\(?\d{1,3}?\)?[-.\s]?\d{1,4}[-.\s]?\d{1,4}[-.\s]?\d{1,9}$`)

var (
	ErrInvalidEmail  = errors.New("invalid email format")
	ErrEmptyEmail    = errors.New("email can not be empty")
	ErrEmptyPassword = errors.New("password cannot be empty")
	ErrEmptyName     = errors.New("name cannot be empty")
	ErrEmptySurname  = errors.New("surname cannot be empty")
	ErrInvalidPhone  = errors.New("invalid phone format")
)

type User struct {
	ID           string `json:"id" db:"id"`
	Email        string `json:"email" db:"email"`
	PasswordHash string `json:"passwordHash" db:"password_hash"`
	Name         string `json:"name" db:"name"`
	Surname      string `json:"surname" db:"surname"`
	Phone        string `json:"phone" db:"phone"`
	Role         int    `json:"role" db:"role"`
	IsVerified   bool   `json:"isVerified" db:"is_verified"`
}

func (u *User) IsClient() bool {
	return u.Role == RoleClient
}

func (u *User) IsAnalycist() bool {
	return u.Role == RoleAnalycist
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

func IsEmailValid(e string) bool {
	if len(e) < 3 || len(e) > 254 {
		return false
	}
	return emailRegex.MatchString(e)
}

func IsPasswordNotEmpty(p string) bool {
	return len(p) > 0
}

func IsNameCorrect(n string) bool {
	return len(n) > 3
}

func IsPhoneValid(p string) bool {
	return phoneRegex.MatchString(p)
}

func NewUser(email, passwordHash, name, surname, phone string) (*User, error) {
	if !IsEmailValid(email) {
		return nil, ErrInvalidEmail
	}

	if !IsNameCorrect(name) {
		return nil, ErrEmptyName
	}

	passwordHash = strings.TrimSpace(passwordHash)
	if !IsPasswordNotEmpty(passwordHash) {
		return nil, ErrEmptyPassword
	}

	if !IsPhoneValid(phone) {
		return nil, ErrInvalidPhone
	}

	return &User{
		ID:           security.GenerateUUID(),
		Email:        email,
		PasswordHash: passwordHash,
		Name:         name,
		Surname:      surname,
		Phone:        phone,
		Role:         1,
		IsVerified:   false,
	}, nil

}

func (u *User) UpdateUser(name, surname, phone string) error {
	trimmedName := strings.TrimSpace(name)
	if !IsNameCorrect(trimmedName) {
		return ErrEmptyName
	}

	trimmedSurname := strings.TrimSpace(surname)

	trimmedPhone := strings.TrimSpace(phone)
	if !IsPhoneValid(trimmedPhone) {
		return ErrInvalidPhone
	}

	u.Name = trimmedName
	u.Surname = trimmedSurname
	u.Phone = phone

	return nil
}

func (u *User) UpdateUserEmail(newEmail string) error {
	trimmedEmail := strings.TrimSpace(newEmail)

	if trimmedEmail == "" {
		return ErrInvalidEmail
	}

	if !IsEmailValid(trimmedEmail) {
		return ErrInvalidEmail
	}

	u.Email = trimmedEmail

	return nil
}
