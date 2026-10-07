package config

import (
	"os"
	"time"
)

type Config struct {
	ResetTokenTTL        time.Duration
	VerificationTokenTTL time.Duration
	AccessTokenTTL       time.Duration
	RefreshTokenTTL      time.Duration
}

func Load() (*Config, error) {
	resetTokenTTL, err := getEnvAsDuration("RESET_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return nil, err
	}
	verificationTokenTTL, err := getEnvAsDuration("VERIFICATION_TOKEN_TTL", 24*time.Hour)
	if err != nil {
		return nil, err
	}
	accessTokenTTL, err := getEnvAsDuration("ACCESS_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return nil, err
	}
	refreshTokenTTL, err := getEnvAsDuration("REFRESH_TOKEN_TTL", 168*time.Hour)
	if err != nil {
		return nil, err
	}

	return &Config{
		ResetTokenTTL:        resetTokenTTL,
		VerificationTokenTTL: verificationTokenTTL,
		AccessTokenTTL:       accessTokenTTL,
		RefreshTokenTTL:      refreshTokenTTL,
	}, err
}

func getEnvAsDuration(key string, fallback time.Duration) (time.Duration, error) {
	if val := os.Getenv(key); val != "" {
		d, err := time.ParseDuration(val)
		if err != nil {
			return 0, err
		}

		return d, nil
	}
	return fallback, nil
}
