package postgre

import "authorization/internal/domain"

type UserModel struct {
	ID           string      `db:"id"`
	Email        string      `db:"email"`
	PasswordHash string      `db:"password_hash"`
	Name         string      `db:"name"`
	Surname      string      `db:"surname"`
	Phone        string      `db:"phone"`
	Role         domain.Role `db:"role"`
	IsVerified   bool        `db:"is_verified"`
	IsActive     bool        `db:"is_active"`
}

func (u *UserModel) ToDomain() *domain.User {
	return &domain.User{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Name:         u.Name,
		Surname:      u.Surname,
		Phone:        u.Phone,
		Role:         u.Role,
		IsVerified:   u.IsVerified,
		IsActive:     u.IsActive,
	}
}

func FromDomain(user *domain.User) *UserModel {
	return &UserModel{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Name:         user.Name,
		Surname:      user.Surname,
		Phone:        user.Phone,
		Role:         user.Role,
		IsVerified:   user.IsVerified,
		IsActive:     user.IsActive,
	}
}
