package rdb

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps go-redis for a minimal ping/ready check.
type Client struct {
	Inner *redis.Client
}

func Connect(ctx context.Context, addr string) (*Client, error) {
	c := redis.NewClient(&redis.Options{Addr: addr, DialTimeout: 5 * time.Second})
	if err := c.Ping(ctx).Err(); err != nil {
		c.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &Client{Inner: c}, nil
}

func (c *Client) Close() error {
	if c == nil || c.Inner == nil {
		return nil
	}
	return c.Inner.Close()
}
