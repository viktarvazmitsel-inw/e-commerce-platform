package token

import (
	"context"
	"crypto/rsa"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/shop/services/notification/internal/domain"
)

type CustomClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Type   string `json:"type"`
	jwt.RegisteredClaims
}

type Parser struct {
	publicKey *rsa.PublicKey
}

func NewParser(publicKey *rsa.PublicKey) (*Parser, error) {
	if publicKey == nil || publicKey.N == nil {
		return nil, fmt.Errorf("public key is nil")
	}
	return &Parser{publicKey: publicKey}, nil
}

func (p *Parser) ParseAccessToken(ctx context.Context, raw string) (domain.Principal, error) {
	if err := ctx.Err(); err != nil {
		return domain.Principal{}, err
	}

	claims, err := p.parse(raw)
	if err != nil {
		return domain.Principal{}, err
	}
	if claims.Type != "access" {
		return domain.Principal{}, fmt.Errorf("token is not an access token")
	}
	if claims.UserID == "" {
		return domain.Principal{}, fmt.Errorf("token user id is empty")
	}

	return domain.Principal{
		UserID: claims.UserID,
		Role:   claims.Role,
	}, nil
}

func (p *Parser) parse(raw string) (*CustomClaims, error) {
	parsed, err := jwt.ParseWithClaims(raw, &CustomClaims{}, func(token *jwt.Token) (any, error) {
		return p.publicKey, nil
	}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}

	claims, ok := parsed.Claims.(*CustomClaims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
