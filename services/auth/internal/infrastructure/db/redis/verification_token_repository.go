package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type VerificationTokenRepository struct {
	redis                *redis.Client
	verificationTokenTTL time.Duration // in seconds
}

func NewVerificationTokenRepository(c *redis.Client, ttl time.Duration) *VerificationTokenRepository {
	return &VerificationTokenRepository{
		redis:                c,
		verificationTokenTTL: ttl,
	}
}

func (r *VerificationTokenRepository) SaveVerificationToken(ctx context.Context, userID, token string) error {
	key := "verification:" + token
	return r.redis.Set(ctx, key, userID, r.verificationTokenTTL).Err()
}

func (r *VerificationTokenRepository) LoadVerificationToken(ctx context.Context, token string) (string, error) {
	key := "verification:" + token
	userID, err := r.redis.Get(ctx, key).Result()

	if err == redis.Nil {
		return "", errors.New("token not found") // Token not found
	} else if err != nil {
		return "", err
	}
	return userID, nil
}

func (r *VerificationTokenRepository) DeleteVerificationToken(ctx context.Context, token string) error {
	key := "verification:" + token
	err := r.redis.Del(ctx, key).Err()
	if err != nil {
		return err
	}
	return nil
}
