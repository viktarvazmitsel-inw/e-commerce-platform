package broker

import (
	"context"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Client struct {
	url       string
	conn      *amqp.Connection
	channel   *amqp.Channel
	mu        sync.RWMutex
	connected bool
}

func NewClient(url string) *Client {
	return &Client{
		url: url,
	}
}

func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := amqp.Dial(c.url)
	if err != nil {
		c.connected = false
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		c.connected = false
		return err
	}

	c.conn = conn
	c.channel = ch
	c.connected = true
	return nil
}

func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected && c.conn != nil && !c.conn.IsClosed()
}

func (c *Client) StartReconnectLoop(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				c.Close()
				return
			default:
				if !c.IsConnected() {
					log.Println("Connecting to RabbitMQ broker...")
					if err := c.Connect(); err != nil {
						log.Printf("Broker connection error: %v. Retrying in 5s...", err)
					} else {
						log.Println("Successfully connected to RabbitMQ broker")
					}
				}
				time.Sleep(5 * time.Second)
			}
		}
	}()
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
	c.connected = false
}
