package redis

import (
	"authorization/internal/domain"
	"authorization/internal/domain/security"
	"authorization/internal/infrastructure/token"
	"authorization/internal/usecase"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenDataExtractor interface {
	GetUserDataByRefreshToken(ctx context.Context, tokenString string) (*token.CustomClaims, error)
}

type SessionRepository struct {
	redisClient        *redis.Client
	refreshTokenTTL    time.Duration
	logger             usecase.Logger
	tokenDataExtractor TokenDataExtractor
}

func getRoleFromString(roleString string) (domain.Role, error) {
	intRole, err := strconv.Atoi(roleString)
	if err != nil {
		return 0, fmt.Errorf("failed to convert role: %w", err)
	}

	if intRole < 0 || intRole > 255 {
		return 0, fmt.Errorf("invalid role")
	}

	return domain.Role(intRole), nil
}

func NewSessionRepository(c *redis.Client, ttl time.Duration, l usecase.Logger, v TokenDataExtractor) *SessionRepository {
	return &SessionRepository{
		redisClient:        c,
		refreshTokenTTL:    ttl,
		logger:             l,
		tokenDataExtractor: v,
	}
}

func (r *SessionRepository) SaveTokenPair(ctx context.Context, userInfo *usecase.UserInfoTokenData, pair domain.TokenPair) error {
	redisKey := "user:sessions:" + userInfo.UserID

	expirationTime := time.Now().Add(r.refreshTokenTTL).Unix()
	refreshTokenHash := security.HashToken(pair.RefreshToken)

	err := r.redisClient.ZAdd(ctx, redisKey, redis.Z{
		Score:  float64(expirationTime),
		Member: refreshTokenHash,
	}).Err()
	if err != nil {
		return err
	}

	if err = ClearUsersExpiredTokens(ctx, r.redisClient, redisKey); err != nil {
		r.logger.Warn(ctx, "Failed to clear expired tokens", "error", err)
	}

	isSet := r.redisClient.Expire(ctx, redisKey, r.refreshTokenTTL)
	if !isSet.Val() {
		r.logger.Warn(ctx, "Failed to set token expiration time in Redis")
	}

	return nil

}

func (r *SessionRepository) GetUserInfoByRefreshToken(ctx context.Context, token string) (*usecase.UserInfoTokenData, error) {
	userData, err := r.tokenDataExtractor.GetUserDataByRefreshToken(ctx, token)
	if err != nil {
		return nil, err
	}
	userID := userData.UserID
	redisKey := "user:sessions:" + userID

	score, err := r.redisClient.ZScore(ctx, redisKey, security.HashToken(token)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("refresh token not found")
		}
		return nil, err
	}
	if score < float64(time.Now().Unix()) {
		return nil, usecase.ErrInvalidRefreshToken
	}

	userRole, err := getRoleFromString(userData.Role)
	if err != nil {
		return nil, err
	}

	return &usecase.UserInfoTokenData{
		UserID: userID,
		Role:   userRole,
	}, nil
}

func (r *SessionRepository) UnsetAndSaveTokenPair(ctx context.Context, oldToken string, userInfo *usecase.UserInfoTokenData, pair domain.TokenPair) error {
	redisKey := "user:sessions:" + userInfo.UserID
	removed, err := r.redisClient.ZRem(ctx, redisKey, security.HashToken(oldToken)).Result()
	if err != nil {
		return err
	}

	if removed == 0 {
		return usecase.ErrInvalidRefreshToken
	}

	err = r.SaveTokenPair(ctx, userInfo, pair)
	if err != nil {
		return err
	}

	return nil
}

func (r *SessionRepository) RevokeAllUserSessions(ctx context.Context, userID string) error {
	redisKey := "user:sessions:" + userID
	_, err := r.redisClient.Del(ctx, redisKey).Result()
	if err != nil {
		return err
	}
	return nil
}

func (r *SessionRepository) RevokeCurrentUserSession(ctx context.Context, userID, token string) error {
	redisKey := "user:sessions:" + userID
	_, err := r.redisClient.ZRem(ctx, redisKey, security.HashToken(token)).Result()
	if err != nil {
		return err
	}
	return nil
}

func (r *SessionRepository) GetRefreshTokenExpireTime(ctx context.Context, token string) (int64, error) {
	tokenHash := security.HashToken(token)
	userData, err := r.tokenDataExtractor.GetUserDataByRefreshToken(ctx, token)
	if err != nil {
		return 0, err
	}

	userID := userData.UserID

	redisKey := "user:sessions:" + userID
	expireTime, err := r.redisClient.ZScore(ctx, redisKey, tokenHash).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("refresh token not found")
		}
		return 0, err
	}

	return int64(expireTime), nil
}

func ClearUsersExpiredTokens(ctx context.Context, client *redis.Client, redisKey string) error {
	now := time.Now().Unix()
	_, err := client.ZRemRangeByScore(ctx, redisKey, "-inf", fmt.Sprintf("%d", now)).Result()
	if err != nil {
		return err
	}

	return nil
}
