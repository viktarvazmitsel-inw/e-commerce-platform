package broker

import (
	"context"
	"fmt"
	"net/url"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RMQConfig struct {
	Host     string
	Port     string
	User     string
	Password string
}

type RMQClient struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func (c *RMQConfig) DSN() string {
	u := &url.URL{
		Scheme: "amqp",
		User:   url.UserPassword(c.User, c.Password),
		Host:   fmt.Sprintf("%s:%s", c.Host, c.Port),
	}

	return u.String()
}

func NewConnection(ctx context.Context, dsn *RMQConfig) (*RMQClient, error) {
	conn, err := amqp.Dial(dsn.DSN())
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &RMQClient{conn: conn, ch: ch}, nil
}

func (c *RMQClient) Close() error {
	if c.ch != nil {
		if err := c.ch.Close(); err != nil {
			c.conn.Close()
			return err
		}
		c.ch = nil
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return err
		}
		c.conn = nil
	}

	return nil
}
