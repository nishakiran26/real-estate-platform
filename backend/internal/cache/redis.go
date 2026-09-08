// Package cache wraps Redis with a fail-soft client: when Redis is unavailable
// every operation degrades to a miss instead of propagating an error, so a Redis
// outage slows the platform down but never takes it offline.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrMiss is returned when a key is absent or Redis is unreachable.
var ErrMiss = errors.New("cache miss")

type Client struct {
	rdb       *redis.Client
	available atomic.Bool
}

func New(addr, password string, db int) *Client {
	c := &Client{rdb: redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c.available.Store(c.rdb.Ping(ctx).Err() == nil)
	return c
}

// Available reports the last known health of the connection.
func (c *Client) Available() bool { return c != nil && c.available.Load() }

// track flips the availability flag based on the outcome of a command so the
// client recovers automatically once Redis comes back.
func (c *Client) track(err error) error {
	if err == nil || errors.Is(err, redis.Nil) {
		c.available.Store(true)
		return err
	}
	c.available.Store(false)
	return err
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if c == nil {
		return "", ErrMiss
	}
	val, err := c.rdb.Get(ctx, key).Result()
	if c.track(err) != nil {
		return "", ErrMiss
	}
	return val, nil
}

func (c *Client) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if c == nil {
		return nil
	}
	return c.track(c.rdb.Set(ctx, key, value, ttl).Err())
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	if c == nil || len(keys) == 0 {
		return nil
	}
	return c.track(c.rdb.Del(ctx, keys...).Err())
}

// DelByPrefix removes every key matching prefix* using SCAN so it never blocks
// the server the way KEYS would on a large keyspace.
func (c *Client) DelByPrefix(ctx context.Context, prefix string) error {
	if c == nil {
		return nil
	}
	var cursor uint64
	for {
		keys, next, err := c.rdb.Scan(ctx, cursor, prefix+"*", 200).Result()
		if c.track(err) != nil {
			return err
		}
		if len(keys) > 0 {
			if err := c.track(c.rdb.Del(ctx, keys...).Err()); err != nil {
				return err
			}
		}
		if next == 0 {
			return nil
		}
		cursor = next
	}
}

// GetJSON unmarshals a cached JSON payload into dest.
func (c *Client) GetJSON(ctx context.Context, key string, dest any) error {
	raw, err := c.Get(ctx, key)
	if err != nil {
		return ErrMiss
	}
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		return ErrMiss
	}
	return nil
}

// SetJSON stores value as JSON. Marshalling failures are swallowed: a cache
// write is never worth failing a request over.
func (c *Client) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) {
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = c.Set(ctx, key, raw, ttl)
}

// Incr increments a counter and applies ttl on first write. Used by the rate
// limiter and OTP attempt tracking.
func (c *Client) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	if c == nil {
		return 0, ErrMiss
	}
	pipe := c.rdb.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	if c.track(err) != nil {
		return 0, ErrMiss
	}
	return incr.Val(), nil
}

// TTL reports the remaining lifetime of a key.
func (c *Client) TTL(ctx context.Context, key string) time.Duration {
	if c == nil {
		return 0
	}
	d, err := c.rdb.TTL(ctx, key).Result()
	if c.track(err) != nil {
		return 0
	}
	return d
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	return c.rdb.Close()
}
