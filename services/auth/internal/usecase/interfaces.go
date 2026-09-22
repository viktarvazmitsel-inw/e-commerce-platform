package usecase

import (
	"authorization/internal/domain"
	"time"
)

type EmailUpdateTokenData struct {
	UserID   string `json:"user_id"`
	NewEmail string `json:"new_email"`
}

type TokenGenerator interface {
	GenerateTokens(ID string) (domain.TokenPair, error)
}

type UserRepository interface {
	IsEmailTaken(email string) bool
	Save(user *domain.User) error
	GetUserByEmail(email string) (*domain.User, error)
	GetUserById(ID string) (*domain.User, error)
	ActivateUser(uuid string) error
}

type TokenRepository interface {
	SaveVerificationToken(userID, token string, ttl time.Duration) error
	LoadVerificationToken(token string) (string, error)
	DeleteVerificationToken(token string) error
	SaveTokenPair(userID string, pair domain.TokenPair) error
	GetUserIDByRefreshToken(token string) (string, error)
	UnsetAndSaveTokenPair(token, userId string, pair domain.TokenPair) error
	SaveEmailUpdateToken(userID, newEmail, token string, ttl time.Duration) error
	LoadEmailUpdateToken(token string) (*EmailUpdateTokenData, error)
	DeleteEmailUpdateToken(token string) error
	RevokeAllUserSessions(userID string) error
}
