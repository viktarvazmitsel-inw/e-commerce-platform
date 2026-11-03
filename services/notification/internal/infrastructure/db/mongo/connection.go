package mongo

import (
	"context"
	"fmt"
	"net/url"

	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Config struct {
	User     string
	Password string
	Host     string
	Port     string
	Database string
}

func (c Config) URI() string {
	u := &url.URL{
		Scheme: "mongodb",
		User:   url.UserPassword(c.User, c.Password),
		Host:   fmt.Sprintf("%s:%s", c.Host, c.Port),
	}
	query := u.Query()
	query.Set("authSource", "admin")
	u.RawQuery = query.Encode()
	return u.String()
}

func NewClient(ctx context.Context, cfg Config) (*drivermongo.Client, error) {
	client, err := drivermongo.Connect(options.Client().ApplyURI(cfg.URI()))
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("unable to ping mongo database: %w", err)
	}

	return client, nil
}
