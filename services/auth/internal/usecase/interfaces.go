package usecase

import (
	"authorization/internal/domain"
	"time"
)

type UserRepository interface {
	IsEmailTaken(email string) bool
	Save(user *domain.User) error
	GetUser(email string) (domain.User, error)
	ActivateUser(uuid string) error
}

type TokenRepository interface {
	SaveVerificationToken(userId string, token string, ttl time.Duration) error
	LoadVerificationToken(token string) (string, error)
	DeleteVerificationToken(token string) error
	SaveTokenPair(userId string, pair domain.TokenPair) error
}
