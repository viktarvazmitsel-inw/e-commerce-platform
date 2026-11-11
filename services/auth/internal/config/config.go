package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ResetTokenTTL        time.Duration
	VerificationTokenTTL time.Duration
	AccessTokenTTL       time.Duration
	RefreshTokenTTL      time.Duration
	DatabaseUser         string
	DatabasePassword     string
	DatabaseName         string
	DatabaseHost         string
	DatabasePort         string
	RedisHost            string
	RedisPort            string
	RedisUser            string
	RedisPassword        string
	RedisDB              int
	TokenPrivateKeyPath  string
	TokenPublicKeyPath   string
	RMQHost              string
	RMQPort              string
	RMQUser              string
	RMQPassword          string
}

func Load() *Config {
	return &Config{
		ResetTokenTTL:        getEnvAsDuration("RESET_TOKEN_TTL", 15*time.Minute),
		VerificationTokenTTL: getEnvAsDuration("VERIFICATION_TOKEN_TTL", 24*time.Hour),
		AccessTokenTTL:       getEnvAsDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:      getEnvAsDuration("REFRESH_TOKEN_TTL", 168*time.Hour),
		DatabaseUser:         os.Getenv("DATABASE_USER"),
		DatabasePassword:     os.Getenv("DATABASE_PASSWORD"),
		DatabaseName:         os.Getenv("DATABASE_NAME"),
		DatabaseHost:         os.Getenv("DATABASE_HOST"),
		DatabasePort:         os.Getenv("DATABASE_PORT"),
		RedisHost:            os.Getenv("REDIS_HOST"),
		RedisPort:            os.Getenv("REDIS_PORT"),
		RedisUser:            os.Getenv("REDIS_USER"),
		RedisPassword:        os.Getenv("REDIS_PASSWORD"),
		RedisDB:              getEnvAsInt("REDIS_DB", 0),
		TokenPrivateKeyPath:  os.Getenv("TOKEN_PRIVATE_KEY_PATH"),
		TokenPublicKeyPath:   os.Getenv("TOKEN_PUBLIC_KEY_PATH"),
		RMQHost:              os.Getenv("RMQ_HOST"),
		RMQPort:              os.Getenv("RMQ_PORT"),
		RMQUser:              os.Getenv("RMQ_USER"),
		RMQPassword:          os.Getenv("RMQ_PASSWORD"),
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

func getEnvAsInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}
