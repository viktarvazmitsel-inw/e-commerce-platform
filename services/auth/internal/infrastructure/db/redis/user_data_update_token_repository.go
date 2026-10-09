package redis

import (
	"authorization/internal/usecase"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type UserDataUpdateTokenRepository struct {
	redisClient   *redis.Client
	resetTokenTTL time.Duration
}

type emailUpdatePayload struct {
	UserID   string `json:"user_id"`
	NewEmail string `json:"new_email"`
}

func NewUserDataUpdateTokenRepository(c *redis.Client, ttl time.Duration) *UserDataUpdateTokenRepository {
	return &UserDataUpdateTokenRepository{
		redisClient:   c,
		resetTokenTTL: ttl,
	}
}

func (r *UserDataUpdateTokenRepository) SaveEmailUpdateToken(ctx context.Context, userID, newEmail, token string) error {
	valueData := emailUpdatePayload{}
	valueData.UserID = userID
	valueData.NewEmail = newEmail

	value, err := json.Marshal(valueData)
	if err != nil {
		return fmt.Errorf("failed to save update token: %w", err)
	}
	valueString := string(value)
	key := "email_update:" + token
	return r.redisClient.Set(ctx, key, valueString, r.resetTokenTTL).Err()
}

func (r *UserDataUpdateTokenRepository) LoadEmailUpdateToken(ctx context.Context, token string) (*usecase.EmailUpdateTokenData, error) {
	key := "email_update:" + token
	valueString, err := r.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	value := []byte(valueString)
	valueData := emailUpdatePayload{}
	if err := json.Unmarshal(value, &valueData); err != nil {
		return nil, fmt.Errorf("failed to load update token: %w", err)
	}

	return &usecase.EmailUpdateTokenData{
		UserID:   valueData.UserID,
		NewEmail: valueData.NewEmail,
	}, nil
}

func (r *UserDataUpdateTokenRepository) SavePasswordResetToken(ctx context.Context, userID, token string) error {
	key := "password_reset:" + token
	return r.redisClient.Set(ctx, key, userID, r.resetTokenTTL).Err()
}

func (r *UserDataUpdateTokenRepository) LoadPasswordResetToken(ctx context.Context, token string) (string, error) {
	key := "password_reset:" + token
	userID, err := r.redisClient.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}

	return userID, nil
}

func (r *UserDataUpdateTokenRepository) DeleteUserDataUpdateToken(ctx context.Context, token string) error {
	err := r.redisClient.Del(ctx, "email_update:"+token, "password_reset"+token).Err()
	if err != nil {
		return err
	}
	return nil
}
