// pkg/limiter/store/redis/store.go
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/seatedro/drl"
)

type RedisStore struct {
	client redis.UniversalClient
	prefix string
}

type Options struct {
	Addresses []string
	KeyPrefix string
}

func NewStore(opts Options) (*RedisStore, error) {
	var client redis.UniversalClient

	client = redis.NewClient(&redis.Options{
		Addr: opts.Addresses[0],
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStore{
		client: client,
		prefix: opts.KeyPrefix,
	}, nil
}

func (s *RedisStore) formatKey(key string) string {
	if s.prefix == "" {
		return fmt.Sprintf("ratelimit:%s", key)
	}
	return fmt.Sprintf("%s:ratelimit:%s", s.prefix, key)
}

// Get retrieves the current state for a key
func (s *RedisStore) Get(ctx context.Context, key string) (*drl.State, error) {
	formattedKey := s.formatKey(key)

	val, err := s.client.Get(ctx, formattedKey).Result()
	if err == redis.Nil {
		return nil, nil // Key doesn't exist
	}
	if err != nil {
		return nil, fmt.Errorf("redis get error: %w", err)
	}

	var state drl.State
	if err := json.Unmarshal([]byte(val), &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// Set updates the state for a key
func (s *RedisStore) Set(ctx context.Context, key string, state *drl.State, ttl time.Duration) error {
	formattedKey := s.formatKey(key)

	value, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := s.client.Set(ctx, formattedKey, value, ttl).Err(); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	return nil
}

// Delete removes a key
func (s *RedisStore) Delete(ctx context.Context, key string) error {
	formattedKey := s.formatKey(key)
	return s.client.Del(ctx, formattedKey).Err()
}

// CleanUp removes expired keys
func (s *RedisStore) CleanUp(ctx context.Context) error {
	return nil // Redis handles TTL automatically
}
