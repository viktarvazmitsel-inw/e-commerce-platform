package token

import (
	"context"
	"crypto/rsa"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

type DataExtractor struct {
	publicKey *rsa.PublicKey
}

func NewDataExtractor(k *rsa.PublicKey) *DataExtractor {
	return &DataExtractor{
		publicKey: k,
	}
}

func (e *DataExtractor) GetUserDataByRefreshToken(ctx context.Context, tokenString string) (*CustomClaims, error) {
	claims, err := e.getClaimsFromToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.Type != "refresh" {
		return nil, fmt.Errorf("token is not a refresh token")
	}

	return claims, nil
}

func (e *DataExtractor) GetDataByAccessToken(ctx context.Context, accessToken string) (*CustomClaims, error) {
	claims, err := e.getClaimsFromToken(accessToken)
	if err != nil {
		return nil, err
	}

	if claims.Type != "access" {
		return nil, fmt.Errorf("token is not an access token")
	}

	return claims, nil
}

func (e *DataExtractor) getClaimsFromToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (any, error) {
		return e.publicKey, nil
	}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithExpirationRequired())

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
