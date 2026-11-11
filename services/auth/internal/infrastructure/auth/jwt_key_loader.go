package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
)

var (
	ErrFailedToReadPrivateKey        = errors.New("failed to read private key")
	ErrFailedToDecodePrivateKey      = errors.New("failed to decode private key")
	ErrFailedToParsePrivateKey       = errors.New("failed to parse private key")
	ErrFailedToReadPublicKey         = errors.New("failed to read public key")
	ErrFailedToDecodePublicKey       = errors.New("failed to decode public key")
	ErrFailedToParsePublicKey        = errors.New("failed to parse public key")
	ErrPublicAndPrivateKeyDoNotMatch = errors.New("public key and private key do not match")
)

func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	privateKey, err := os.ReadFile(path)
	if err != nil {
		return nil, ErrFailedToReadPrivateKey
	}

	block, _ := pem.Decode(privateKey)
	if block == nil {
		return nil, ErrFailedToDecodePrivateKey
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, ErrFailedToParsePrivateKey
		}

		rsaKey, ok := parsedKey.(*rsa.PrivateKey)
		if !ok {
			return nil, ErrFailedToParsePrivateKey
		}

		return rsaKey, nil
	}

	return key, nil
}

func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	publicKeyString, err := os.ReadFile(path)
	if err != nil {
		return nil, ErrFailedToReadPublicKey
	}

	block, _ := pem.Decode(publicKeyString)
	if block == nil {
		return nil, ErrFailedToDecodePublicKey
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, ErrFailedToParsePublicKey
	}

	publicKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, ErrFailedToParsePublicKey
	}
	return publicKey, nil
}

func ValidateKeys(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey) error {
	if privateKey.N.Cmp(publicKey.N) != 0 {
		return ErrPublicAndPrivateKeyDoNotMatch
	}

	return nil
}
