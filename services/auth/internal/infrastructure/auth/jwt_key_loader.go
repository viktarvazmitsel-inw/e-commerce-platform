package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

type JWTCacher interface {
	SavePublicKey(key *rsa.PublicKey) error
}

type JwtKeyLoader struct {
	cache JWTCacher
}

func NewJwtKeyLoader(cache JWTCacher) *JwtKeyLoader {
	return &JwtKeyLoader{cache: cache}
}

func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	privateKey, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	block, _ := pem.Decode(privateKey)
	if block == nil {
		return nil, fmt.Errorf("failed to decode private key")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		return parsedKey.(*rsa.PrivateKey), nil
	}

	return key, nil
}

func (l *JwtKeyLoader) LoadPublicKey(path string) (*rsa.PublicKey, error) {
	publicKey, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	block, _ := pem.Decode(publicKey)
	if block == nil {
		return nil, fmt.Errorf("failed to decode public key")
	}

	key, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	err = l.cache.SavePublicKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to save public key to cache: %w", err)
	}

	return key, nil
}
