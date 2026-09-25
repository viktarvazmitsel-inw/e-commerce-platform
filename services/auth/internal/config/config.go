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

func Load() *Config {
	return &Config{
		ResetTokenTTL:        getEnvAsDuration("RESET_TOKEN_TTL", 15*time.Minute),
		VerificationTokenTTL: getEnvAsDuration("VERIFICATION_TOKEN_TTL", 24*time.Hour),
		AccessTokenTTL:       getEnvAsDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:      getEnvAsDuration("REFRESH_TOKEN_TTL", 162*time.Hour),
	}
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return fallback
}
