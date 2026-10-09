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
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("wrong token sign algoritm: %v", token.Header["alg"])
		}
		return e.publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	if claims.Type != "refresh" {
		return nil, fmt.Errorf("token is not a refresh token")
	}

	return claims, nil
}
