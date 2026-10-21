package redis

import (
	"context"
	"crypto/rsa"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type JwtKeyRepository struct {
	redis *redis.Client
}

func NewJwtKeyRepository(redis *redis.Client) *JwtKeyRepository {
	return &JwtKeyRepository{redis: redis}
}

func (r *JwtKeyRepository) SavePublicKey(key *rsa.PublicKey) error {
	storageKey := fmt.Sprintf("jwt:public_key:%s", key.N.String())

	_, err := r.redis.Set(context.Background(), storageKey, key, 0).Result()
	if err != nil {
		return fmt.Errorf("failed to save public key to cache: %w", err)
	}

	return nil
}
