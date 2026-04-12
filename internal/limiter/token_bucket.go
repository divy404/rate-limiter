package limiter

import (
    "context"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

type TokenBucketLimiter struct {
    rdb      *redis.Client
    capacity int64   // max tokens in bucket
    refillRate int64 // tokens added per second
}

func NewTokenBucketLimiter(rdb *redis.Client, capacity, refillRate int64) *TokenBucketLimiter {
    return &TokenBucketLimiter{
        rdb:        rdb,
        capacity:   capacity,
        refillRate: refillRate,
    }
}

// Allow checks if a request from 'clientID' is allowed
func (t *TokenBucketLimiter) Allow(ctx context.Context, clientID string) (bool, error) {
    key := fmt.Sprintf("token_bucket:%s", clientID)
    now := time.Now().Unix()

    // Lua script runs atomically in Redis — no race conditions
    script := redis.NewScript(`
        local key = KEYS[1]
        local capacity = tonumber(ARGV[1])
        local refill_rate = tonumber(ARGV[2])
        local now = tonumber(ARGV[3])

        -- get current state
        local bucket = redis.call("HMGET", key, "tokens", "last_refill")
        local tokens = tonumber(bucket[1])
        local last_refill = tonumber(bucket[2])

        -- first request for this client
        if tokens == nil then
            tokens = capacity
            last_refill = now
        end

        -- refill tokens based on time passed
        local elapsed = now - last_refill
        tokens = math.min(capacity, tokens + elapsed * refill_rate)

        -- check if request is allowed
        if tokens >= 1 then
            tokens = tokens - 1
            redis.call("HMSET", key, "tokens", tokens, "last_refill", now)
            redis.call("EXPIRE", key, 3600)
            return 1  -- allowed
        else
            redis.call("HMSET", key, "tokens", tokens, "last_refill", now)
            return 0  -- denied
        end
    `)

    result, err := script.Run(ctx, t.rdb, []string{key},
        t.capacity, t.refillRate, now).Int()
    if err != nil {
        return false, fmt.Errorf("redis error: %w", err)
    }

    return result == 1, nil
}