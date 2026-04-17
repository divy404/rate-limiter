package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type SlidingWindowLimiter struct {
	rdb        *redis.Client
	limit      int64
	windowSize time.Duration
}

func NewSlidingWindowLimiter(rdb *redis.Client, limit int64, windowSize time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		rdb:        rdb,
		limit:      limit,
		windowSize: windowSize,
	}
}

func (s *SlidingWindowLimiter) Allow(ctx context.Context, clientID string) (bool, error) {
	key := fmt.Sprintf("sliding_window:%s", clientID)
	now := time.Now()
	windowStart := now.Add(-s.windowSize).UnixMicro()
	nowMicro := now.UnixMicro()

	script := redis.NewScript(`
		local key = KEYS[1]
		local window_start = tonumber(ARGV[1])
		local now = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local window_seconds = tonumber(ARGV[4])

		-- remove entries older than our window
		redis.call("ZREMRANGEBYSCORE", key, "-inf", window_start)

		local count = redis.call("ZCARD", key)

		if count < limit then
			redis.call("ZADD", key, now, now .. ":" .. math.random(1000000))
			redis.call("EXPIRE", key, window_seconds)
			return 1
		else
			return 0
		end
	`)

	windowSeconds := int(s.windowSize.Seconds())
	result, err := script.Run(ctx, s.rdb, []string{key},
		windowStart, nowMicro, s.limit, windowSeconds).Int()
	if err != nil {
		return false, fmt.Errorf("redis error: %w", err)
	}

	return result == 1, nil
}