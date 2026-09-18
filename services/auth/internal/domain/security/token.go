package security

import (
	"crypto/rand"
	"encoding/hex"
	"uuid"
)

func GenerateEmailVerificationToken(uuid string) (string, error) {
	bytes := make([]byte, 30)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func GenerateUUID() string {
	return uuid.NewV7().String()
}
