package token

import (
	"authorization/internal/domain"
	"context"
	"crypto/rsa"
	"fmt"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"
)

type JWTGenerator struct {
	privateKey      *rsa.PrivateKey
	refreshTokenTTL time.Duration
	accessTokenTTL  time.Duration
	now             func() time.Time
}

func NewJWTGenerator(privateKey *rsa.PrivateKey, refreshTTL, accessTTL time.Duration) (*JWTGenerator, error) {
	return &JWTGenerator{
		privateKey:      privateKey,
		refreshTokenTTL: refreshTTL,
		accessTokenTTL:  accessTTL,
		now:             time.Now,
	}, nil
}

func (c *JWTGenerator) GenerateTokens(
	ctx context.Context,
	ID string,
	role domain.Role,
) (domain.TokenPair, error) {
	issuedAt := c.now()
	refreshExpire := c.now().Add(c.refreshTokenTTL)
	accessExpire := c.now().Add(c.accessTokenTTL)

	refreshClaims := CustomClaims{
		UserID:    ID,
		Role:      role.String(),
		Type:      "refresh",
		Subject:   ID,
		IssuedAt:  jwt.NewNumericDate(issuedAt),
		ExpiresAt: jwt.NewNumericDate(refreshExpire),
		ID:        uuid.New().String(),
	}

	refresh := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)
	signedRefresh, err := refresh.SignedString(c.privateKey)
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	accessClaims := CustomClaims{
		UserID:    ID,
		Role:      role.String(),
		Type:      "access",
		Subject:   ID,
		IssuedAt:  jwt.NewNumericDate(issuedAt),
		ExpiresAt: jwt.NewNumericDate(accessExpire),
		ID:        uuid.New().String(),
	}

	access := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	signedAccess, err := access.SignedString(c.privateKey)
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("failed to sign access token: %w", err)
	}

	return domain.TokenPair{
		AccessToken:           signedAccess,
		RefreshToken:          signedRefresh,
		AccessTokenExpiresAt:  accessExpire,
		RefreshTokenExpiresAt: refreshExpire,
	}, nil
}
