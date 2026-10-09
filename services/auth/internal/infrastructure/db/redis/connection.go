package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DB       int
}

func NewRedisClient(ctx context.Context, cfg *RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Host + ":" + cfg.Port,
		Username:     cfg.User,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     10,              // Adjust the pool size as needed
		MinIdleConns: 5,               // Adjust the minimum idle connections as needed
		DialTimeout:  5 * time.Second, // Adjust the dial timeout as needed
		ReadTimeout:  3 * time.Second, // Adjust the read timeout as needed
		WriteTimeout: 3 * time.Second, // Adjust the write timeout as needed
	})

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return client, nil
}
