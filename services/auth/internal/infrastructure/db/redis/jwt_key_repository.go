package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type JwtKeyRepository struct {
	redis *redis.Client
}

func NewJwtKeyRepository(redis *redis.Client) *JwtKeyRepository {
	return &JwtKeyRepository{redis: redis}
}

func (r *JwtKeyRepository) SavePublicKey(key string) error {
	storageKey := "jwt:public_key:current"

	_, err := r.redis.Set(context.Background(), storageKey, key, 0).Result()
	if err != nil {
		return fmt.Errorf("failed to save public key to cache: %w", err)
	}

	return nil
}
