package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type FixedWindowLimiter struct {
	rdb        *redis.Client
	limit      int64
	windowSize time.Duration
}

func NewFixedWindowLimiter(rdb *redis.Client, limit int64, windowSize time.Duration) *FixedWindowLimiter {
	return &FixedWindowLimiter{
		rdb:        rdb,
		limit:      limit,
		windowSize: windowSize,
	}
}

func (f *FixedWindowLimiter) Allow(ctx context.Context, clientID string) (bool, error) {
	// Key includes window timestamp — all requests in same window share it
	windowStart := time.Now().Truncate(f.windowSize).Unix()
	key := fmt.Sprintf("fixed_window:%s:%d", clientID, windowStart)

	script := redis.NewScript(`
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local window_seconds = tonumber(ARGV[2])

		local count = redis.call("INCR", key)

		if count == 1 then
			redis.call("EXPIRE", key, window_seconds)
		end

		if count <= limit then
			return 1
		else
			return 0
		end
	`)

	windowSeconds := int(f.windowSize.Seconds())
	result, err := script.Run(ctx, f.rdb, []string{key},
		f.limit, windowSeconds).Int()
	if err != nil {
		return false, fmt.Errorf("redis error: %w", err)
	}

	return result == 1, nil
}