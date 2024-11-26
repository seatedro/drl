package drl

import (
	"context"
	"fmt"
	"time"
)

// RateLimiter provides rate limiting functionality using the token bucket algorithm
type RateLimiter interface {
	// Allow checks if a request should be allowed
	// Returns: allowed, remaining tokens, time until reset, error
	Allow(ctx context.Context, key string) (bool, int, time.Duration, error)

	// Reset resets the rate limiter for a key
	Reset(ctx context.Context, key string) error
}

// Config holds the configuration for the rate limiter
type Config struct {
	// Rate is how many requests are allowed per window
	Rate int

	// Window is the time window for rate limiting
	Window time.Duration

	// BurstSize allows for temporary bursts of requests (optional)
	// If not set, defaults to Rate
	BurstSize int
}

// Store defines the interface for the backing storage
type Store interface {
	// Get retrieves the current state for a key
	Get(ctx context.Context, key string) (*State, error)

	// Set updates the state for a key
	Set(ctx context.Context, key string, state *State, ttl time.Duration) error

	// Delete removes a key
	Delete(ctx context.Context, key string) error
}

// State represents the current state of rate limiting for a key
type State struct {
	// Tokens represents the current number of available tokens
	Tokens float64 `json:"tokens"`

	// LastRefill is the last time tokens were added
	LastRefill time.Time `json:"last_refill"`
}

// limiter implements the RateLimiter interface
type limiter struct {
	config Config
	store  Store
}

// New creates a new rate limiter
func New(config Config, store Store) (RateLimiter, error) {
	if config.Rate <= 0 {
		return nil, fmt.Errorf("rate must be positive")
	}
	if config.Window <= 0 {
		return nil, fmt.Errorf("window must be positive")
	}
	if config.BurstSize < config.Rate {
		config.BurstSize = config.Rate
	}

	return &limiter{
		config: config,
		store:  store,
	}, nil
}

// Allow implements the token bucket algorithm
func (l *limiter) Allow(ctx context.Context, key string) (bool, int, time.Duration, error) {
	state, err := l.store.Get(ctx, key)
	if err != nil {
		return false, 0, 0, fmt.Errorf("failed to get state: %w", err)
	}

	now := time.Now()

	// If this is a new key, initialize it with full tokens
	if state == nil {
		state = &State{
			Tokens:     float64(l.config.BurstSize),
			LastRefill: now,
		}
	}

	// Calculate tokens to add based on time passed
	timePassed := now.Sub(state.LastRefill)
	tokensToAdd := (float64(l.config.Rate) / float64(l.config.Window)) * float64(timePassed)

	// Add new tokens, not exceeding burst size
	newTokens := state.Tokens + tokensToAdd
	if newTokens > float64(l.config.BurstSize) {
		newTokens = float64(l.config.BurstSize)
	}

	// Check if we can fulfill the request
	allowed := newTokens >= 1.0
	if allowed {
		newTokens--
	}

	// Update state
	state.Tokens = newTokens
	state.LastRefill = now

	// Calculate reset time (time until bucket is full)
	tokensNeeded := float64(l.config.BurstSize) - newTokens
	refillRate := float64(l.config.Rate) / float64(l.config.Window)
	resetDuration := time.Duration(tokensNeeded / refillRate)

	// Store updated state with TTL of 2x window to ensure cleanup of unused keys
	if err := l.store.Set(ctx, key, state, l.config.Window*2); err != nil {
		return false, 0, 0, fmt.Errorf("failed to store state: %w", err)
	}

	return allowed, int(newTokens), resetDuration, nil
}

// Reset implements the RateLimiter interface
func (l *limiter) Reset(ctx context.Context, key string) error {
	return l.store.Delete(ctx, key)
}
