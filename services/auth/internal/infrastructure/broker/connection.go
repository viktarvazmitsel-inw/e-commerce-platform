package broker

import (
	"context"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RMQConfig struct {
	Host     string
	Port     string
	Username string
	Password string
}

type RMQClient struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	mu   sync.Mutex
}

func (c *RMQConfig) DSN() string {
	return "amqp://" + c.Username + ":" + c.Password + "@" + c.Host + ":" + c.Port
}

func NewConnection(ctx context.Context, dsn *RMQConfig) (*RMQClient, error) {
	conn, err := amqp.Dial(dsn.DSN())
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	return &RMQClient{conn: conn, ch: ch}, nil
}

func (c *RMQClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ch != nil {
		if err := c.ch.Close(); err != nil {
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
