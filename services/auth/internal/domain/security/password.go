package security

import "golang.org/x/crypto/bcrypt"

func HashPassword(p string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)

	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func IsPasswordValid(password, sample string) (bool, error) {
	passwordHash, err := HashPassword(password)
	if err != nil {
		return false, err
	}

	return passwordHash == sample, nil
}
