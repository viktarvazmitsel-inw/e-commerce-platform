package search

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
)

type Client struct {
	es *elasticsearch.Client
}

func NewClient(address string) (*Client, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{address},
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	client := &Client{es: es}
	return client, nil
}

func (c *Client) Ping(ctx context.Context) bool {
	if c == nil || c.es == nil {
		return false
	}
	res, err := c.es.Ping(c.es.Ping.WithContext(ctx))
	if err != nil {
		return false
	}
	defer res.Body.Close()
	return res.StatusCode == http.StatusOK
}

func (c *Client) Search(ctx context.Context, query string) ([]interface{}, int, error) {
	if c == nil || c.es == nil {
		return []interface{}{}, 0, nil
	}

	var buf strings.Builder
	if query == "" {
		buf.WriteString(`{"query":{"match_all":{}}}`)
	} else {
		buf.WriteString(fmt.Sprintf(`{"query":{"multi_match":{"query":"%s"}}}`, query))
	}

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithBody(strings.NewReader(buf.String())),
		c.es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		log.Printf("Elasticsearch search error: %v", err)
		return []interface{}{}, 0, nil
	}
	defer res.Body.Close()

	if res.IsError() {
		return []interface{}{}, 0, nil
	}

	return []interface{}{}, 0, nil
}

func (c *Client) ConnectWithRetry(ctx context.Context, maxRetries int, delay time.Duration) {
	for i := 0; i < maxRetries; i++ {
		if c.Ping(ctx) {
			log.Println("Successfully connected to Elasticsearch")
			return
		}
		log.Printf("Waiting for Elasticsearch (attempt %d/%d)...", i+1, maxRetries)
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
	}
	log.Println("Could not establish immediate connection to Elasticsearch, continuing in background...")
}
