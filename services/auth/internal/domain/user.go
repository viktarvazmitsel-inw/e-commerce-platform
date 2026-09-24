package domain

import (
	"authorization/internal/domain/security"
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
var phoneRegex = regexp.MustCompile(`^\+?\d{1,4}?[-.\s]?\(?\d{1,3}?\)?[-.\s]?\d{1,4}[-.\s]?\d{1,4}[-.\s]?\d{1,9}$`)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	Surname      string
	Phone        string
	Role         Role
	IsVerified   bool
	IsActive     bool
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
	return len(n) > 1
}

func IsPhoneValid(p string) bool {
	return phoneRegex.MatchString(p)
}

func NewUser(email, passwordHash, name, surname, phone string) (*User, error) {
	if !IsEmailValid(email) {
		return nil, ErrInvalidEmail
	}
	trimmedName := strings.TrimSpace(name)
	if !IsNameCorrect(trimmedName) {
		return nil, ErrEmptyName
	}

	trimmedSurname := strings.TrimSpace(surname)

	passwordHash = strings.TrimSpace(passwordHash)
	if !IsPasswordNotEmpty(passwordHash) {
		return nil, ErrEmptyPassword
	}

	trimmedPhone := strings.TrimSpace(phone)
	if !IsPhoneValid(trimmedPhone) {
		return nil, ErrInvalidPhone
	}

	return &User{
		ID:           security.GenerateUUID(),
		Email:        email,
		PasswordHash: passwordHash,
		Name:         trimmedName,
		Surname:      trimmedSurname,
		Phone:        trimmedPhone,
		Role:         1,
		IsVerified:   false,
		IsActive:     true,
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
	u.Phone = trimmedPhone

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

func (u *User) UpdateUserPassword(newPasswordHash string) error {
	trimmedPasswordHash := strings.TrimSpace(newPasswordHash)

	if trimmedPasswordHash == "" {
		return ErrEmptyPassword
	}

	u.PasswordHash = trimmedPasswordHash

	return nil
}

func (u *User) Can(permission Permission) bool {
	return u.Role.HasPermission(permission)
}

func (u *User) RequirePermission(permission Permission) error {
	if !u.Can(permission) {
		return ErrPermissionDenied
	}
	return nil
}

func (u *User) SetRole(role Role) error {
	if role == RoleAdmin {
		return ErrUnableToSetAdmin
	}

	if role == u.Role {
		return ErrUnchangedRole
	}

	u.Role = role

	return nil
}

func (u *User) EnsureActive() error {
	if !u.IsActive {
		return ErrUserDeactivated
	}

	return nil
}

func (u *User) Deactivate() {
	u.IsActive = false
}

func (u *User) VerifyEmail() {
	u.IsVerified = true
}
