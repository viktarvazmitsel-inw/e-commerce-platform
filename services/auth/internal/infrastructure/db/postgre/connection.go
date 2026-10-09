package postgre

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	DBName   string
}

func (c *PostgresConfig) DSN() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   fmt.Sprintf("%s:%s", c.Host, c.Port),
		Path:   "/" + strings.TrimPrefix(c.DBName, "/"),
	}
	return u.String()
}

func NewPostgresPool(ctx context.Context, dsn *PostgresConfig) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(dsn.DSN())
	if err != nil {
		return nil, err
	}

	poolConfig.MaxConns = 20                      // Set the maximum number of connections in the pool
	poolConfig.MinConns = 5                       // Set the minimum number of connections in the pool
	poolConfig.MaxConnLifetime = 30 * time.Minute // Set the maximum lifetime of a connection (0 means no limit)
	poolConfig.MaxConnIdleTime = 5 * time.Minute  // Set the maximum idle time for a connection (0 means no limit)

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping postgres database: %w", err)
	}

	return pool, nil
}
