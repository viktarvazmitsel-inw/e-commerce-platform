package redis

import (
	"authorization/internal/domain"
	"authorization/internal/domain/security"
	"authorization/internal/infrastructure/token"
	"authorization/internal/usecase"
	"context"
	"errors"
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

	isSet, err := r.redisClient.Expire(ctx, redisKey, r.refreshTokenTTL).Result()
	if err != nil {
		r.logger.Warn(ctx, "Failed to set token expiration time in Redis", "error", err)
		return err
	}
	if !isSet {
		r.logger.Warn(ctx, "Failed to set token expiration time in Redis", "error", err)
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

var unsetAndSaveTokenScript = redis.NewScript(`
local oldScore = redis.call('ZSCORE', KEYS[1], ARGV[1])
if not oldScore then
    return 0
end

local ok, err = pcall(redis.call, 'ZADD', KEYS[1], ARGV[2], ARGV[3])
if not ok then
    return redis.error_reply(tostring(err))
end

redis.call('ZREM', KEYS[1], ARGV[1])
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', ARGV[4])
redis.call('EXPIRE', KEYS[1], ARGV[5])

return 1
`)

func (r *SessionRepository) UnsetAndSaveTokenPair(ctx context.Context, oldToken string, userInfo *usecase.UserInfoTokenData, pair domain.TokenPair) error {
	redisKey := "user:sessions:" + userInfo.UserID
	now := time.Now()
	expirationTime := now.Add(r.refreshTokenTTL).Unix()
	refreshTokenHash := security.HashToken(pair.RefreshToken)
	oldTokenHash := security.HashToken(oldToken)
	ttlSeconds := int64(r.refreshTokenTTL.Seconds())
	if r.refreshTokenTTL > 0 && ttlSeconds == 0 {
		ttlSeconds = 1
	}

	res, err := unsetAndSaveTokenScript.Run(
		ctx,
		r.redisClient,
		[]string{redisKey},
		oldTokenHash,
		expirationTime,
		refreshTokenHash,
		now.Unix(),
		ttlSeconds,
	).Int()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return usecase.ErrInvalidRefreshToken
		}
		return err
	}

	if res == 0 {
		return usecase.ErrInvalidRefreshToken
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
