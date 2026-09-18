package usecase

import (
	"authorization/internal/domain"
	"time"
)

type TokenGenerator interface {
	GenerateTokens(id string) (domain.TokenPair, error)
}

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
	GetUserIDByRefreshToken(token string) (string, error)
	UnsetAndSaveTokenPair(token, userId string, pair domain.TokenPair) error
}
