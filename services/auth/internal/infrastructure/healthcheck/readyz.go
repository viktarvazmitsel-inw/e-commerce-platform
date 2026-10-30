package healthcheck

import (
	"authorization/internal/infrastructure/broker"
	"context"
	"sync/atomic"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type ReadyzHealthcheck struct {
	db        *pgxpool.Pool
	redis     *redis.Client
	broker    *broker.RMQClient
	isHealthy atomic.Bool
}

func NewReadyzHealthcheck(db *pgxpool.Pool, redis *redis.Client, broker *broker.RMQClient) *ReadyzHealthcheck {
	return &ReadyzHealthcheck{
		db:     db,
		redis:  redis,
		broker: broker,
	}
}

func (h *ReadyzHealthcheck) Check(ctx context.Context) {
	healthy := true

	if err := h.db.Ping(ctx); err != nil {
		healthy = false
	}

	if err := h.redis.Ping(ctx).Err(); err != nil {
		healthy = false
	}

	if h.broker.IsClosed() {
		healthy = false
	}

	h.isHealthy.Store(healthy)
}

func (h *ReadyzHealthcheck) IsHealthy() bool {
	return h.isHealthy.Load()
}
