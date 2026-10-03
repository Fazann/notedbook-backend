package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Set stores value under key for ttl.
func Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return RedisClient.Set(ctx, key, value, ttl).Err()
}

// Get returns the value of key; found is false when the key does not exist or has expired.
func Get(ctx context.Context, key string) (value string, found bool, err error) {
	value, err = RedisClient.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	return value, err == nil, err
}

// GetDel returns the value of key and deletes it in one step, so it can be read only once.
func GetDel(ctx context.Context, key string) (value string, found bool, err error) {
	value, err = RedisClient.GetDel(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	return value, err == nil, err
}

// Del deletes keys.
func Del(ctx context.Context, keys ...string) error {
	return RedisClient.Del(ctx, keys...).Err()
}

// Store exposes Get, Set and Del as methods, so a service can depend on a small cache interface
// and use a fake in tests.
type Store struct{}

func (Store) Get(ctx context.Context, key string) (string, bool, error) { return Get(ctx, key) }

func (Store) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return Set(ctx, key, value, ttl)
}

func (Store) Del(ctx context.Context, keys ...string) error { return Del(ctx, keys...) }
