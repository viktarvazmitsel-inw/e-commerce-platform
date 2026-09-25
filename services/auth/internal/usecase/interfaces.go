package usecase

import (
	"authorization/internal/domain"
	"context"
)

type EmailUpdateTokenData struct {
	UserID   string `json:"user_id"`
	NewEmail string `json:"new_email"`
}

type PasswordResetTokenData struct {
	UserID string `json:"user_id"`
}

type UserInfoTokenData struct {
	UserID string      `json:"user_id"`
	Role   domain.Role `json:"role"`
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(password, hash string) bool
}

type TokenGenerator interface {
	GenerateTokens(ctx context.Context, ID string, role domain.Role) (domain.TokenPair, error)
}

type UserRepository interface {
	IsEmailTaken(ctx context.Context, email string) (bool, error)
	Save(ctx context.Context, user *domain.User) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByID(ctx context.Context, ID string) (*domain.User, error)
}

type VerificationTokenRepository interface {
	SaveVerificationToken(ctx context.Context, userID, token string) error
	LoadVerificationToken(ctx context.Context, token string) (string, error)
	DeleteVerificationToken(ctx context.Context, token string) error
}

type UserDataUpdateTokenRepository interface {
	SaveEmailUpdateToken(ctx context.Context, userID, newEmail, token string) error
	LoadEmailUpdateToken(ctx context.Context, token string) (*EmailUpdateTokenData, error)
	SavePasswordResetToken(ctx context.Context, userID, token string) error
	LoadPasswordResetToken(ctx context.Context, token string) (*PasswordResetTokenData, error)
	DeleteUserDataUpdateToken(ctx context.Context, token string) error
}

type SessionRepository interface {
	SaveTokenPair(ctx context.Context, userID string, pair domain.TokenPair) error
	GetUserInfoByRefreshToken(ctx context.Context, token string) (*UserInfoTokenData, error)
	UnsetAndSaveTokenPair(ctx context.Context, token, userId string, pair domain.TokenPair) error
	RevokeAllUserSessions(ctx context.Context, userID string) error
	RevokeCurrentUserSession(ctx context.Context, userID, token string) error
}

type EmailSender interface {
	SendVerificationEmail(ctx context.Context, email, token string) error
	SendPasswordResetEmail(ctx context.Context, email, token string) error
}
