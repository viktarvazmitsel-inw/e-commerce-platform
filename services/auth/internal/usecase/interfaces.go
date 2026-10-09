package usecase

import (
	"authorization/internal/domain"
	"context"
)

type EmailUpdateTokenData struct {
	UserID   string
	NewEmail string
}

type UserInfoTokenData struct {
	UserID string
	Role   domain.Role
}

type Logger interface {
	Warn(ctx context.Context, msg string, keysAndValues ...any)
	Error(ctx context.Context, msg string, keysAndValues ...any)
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
	Create(ctx context.Context, user *domain.User) error
	Save(ctx context.Context, user *domain.User) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByID(ctx context.Context, ID string) (*domain.User, error)
	GetUserList(ctx context.Context, page, quantity int) ([]domain.User, int, error)
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
	LoadPasswordResetToken(ctx context.Context, token string) (string, error)
	DeleteUserDataUpdateToken(ctx context.Context, token string) error
}

type SessionRepository interface {
	SaveTokenPair(ctx context.Context, userInfo *UserInfoTokenData, pair domain.TokenPair) error
	GetUserInfoByRefreshToken(ctx context.Context, token string) (*UserInfoTokenData, error)
	UnsetAndSaveTokenPair(ctx context.Context, oldToken string, userInfo *UserInfoTokenData, pair domain.TokenPair) error
	RevokeAllUserSessions(ctx context.Context, userID string) error
	RevokeCurrentUserSession(ctx context.Context, userID, token string) error
	GetRefreshTokenExpireTime(ctx context.Context, token string) (int64, error)
}

type EmailSender interface {
	SendVerificationEmail(ctx context.Context, email, token string) error
	SendPasswordResetEmail(ctx context.Context, email, token string) error
}
