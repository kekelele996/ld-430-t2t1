package client

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient wraps go-redis with helpers used by rate limiter and cache.
type RedisClient struct {
	Client *redis.Client
}

// NewRedisClient connects to Redis.
func NewRedisClient(ctx context.Context, url, password string) (*RedisClient, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	if password != "" {
		opts.Password = password
	}
	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &RedisClient{Client: client}, nil
}

// IncrWithTTL increments a key, setting TTL on first creation. Returns current count.
func (r *RedisClient) IncrWithTTL(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	count, err := r.Client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("redis incr %s: %w", key, err)
	}
	if count == 1 {
		r.Client.Expire(ctx, key, ttl)
	}
	return count, nil
}

// Get fetches a string value by key.
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key).Result()
}

// Set stores a value with TTL.
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.Client.Set(ctx, key, value, ttl).Err()
}

// IncrDailyCredentialUsage increments the current UTC-day call counter for a team
// credential, expiring the key at UTC midnight. It reports whether the counter
// has passed the credential's daily limit.
func (r *RedisClient) IncrDailyCredentialUsage(ctx context.Context, credentialID string, limit int) (int64, bool, error) {
	now := time.Now().UTC()
	day := now.Format("20060102")
	key := fmt.Sprintf("credential:quota:%s:%s", credentialID, day)
	count, err := r.Client.Incr(ctx, key).Result()
	if err != nil {
		return 0, false, fmt.Errorf("redis incr %s: %w", key, err)
	}
	if count == 1 {
		r.Client.Expire(ctx, key, durationUntilUTCMidnight(now))
	}
	return count, count > int64(limit), nil
}

// durationUntilUTCMidnight returns the duration from t to the next UTC midnight.
func durationUntilUTCMidnight(t time.Time) time.Duration {
	next := t.Add(24 * time.Hour).Truncate(24 * time.Hour)
	d := next.Sub(t)
	if d <= 0 {
		return 24 * time.Hour
	}
	return d
}

// Close closes the underlying client.
func (r *RedisClient) Close() error {
	return r.Client.Close()
}
