package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"uuid"
)

func GenerateUserDataUpdateToken() (string, error) {
	bytes := make([]byte, 20)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func HashToken(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(hash[:])
}

func GenerateUUID() string {
	return uuid.NewV7().String()
}
