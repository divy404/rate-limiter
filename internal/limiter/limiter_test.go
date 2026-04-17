package limiter

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func newTestRedis(t *testing.T) *redis.Client {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6389"})
	rdb.FlushDB(context.Background()) // clean slate for each test
	return rdb
}

func TestTokenBucket(t *testing.T) {
	rdb := newTestRedis(t)
	defer rdb.Close()

	limiter := NewTokenBucketLimiter(rdb, 5, 1)
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		allowed, err := limiter.Allow(ctx, "tb-user")
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("TokenBucket Request %d: allowed=%v\n", i, allowed)
		if !allowed {
			t.Errorf("Request %d should be allowed", i)
		}
	}

	allowed, _ := limiter.Allow(ctx, "tb-user")
	if allowed {
		t.Error("6th request should be denied")
	}
	fmt.Println("TokenBucket Request 6: denied ✓")
}

func TestFixedWindow(t *testing.T) {
	rdb := newTestRedis(t)
	defer rdb.Close()

	limiter := NewFixedWindowLimiter(rdb, 3, time.Minute)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		allowed, err := limiter.Allow(ctx, "fw-user")
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("FixedWindow Request %d: allowed=%v\n", i, allowed)
		if !allowed {
			t.Errorf("Request %d should be allowed", i)
		}
	}

	allowed, _ := limiter.Allow(ctx, "fw-user")
	if allowed {
		t.Error("4th request should be denied")
	}
	fmt.Println("FixedWindow Request 4: denied ✓")
}

func TestSlidingWindow(t *testing.T) {
	rdb := newTestRedis(t)
	defer rdb.Close()

	limiter := NewSlidingWindowLimiter(rdb, 3, time.Minute)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		allowed, err := limiter.Allow(ctx, "sw-user")
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("SlidingWindow Request %d: allowed=%v\n", i, allowed)
		if !allowed {
			t.Errorf("Request %d should be allowed", i)
		}
	}

	allowed, _ := limiter.Allow(ctx, "sw-user")
	if allowed {
		t.Error("4th request should be denied")
	}
	fmt.Println("SlidingWindow Request 4: denied ✓")
}

func TestAllImplementLimiterInterface(t *testing.T) {
	rdb := newTestRedis(t)
	defer rdb.Close()

	limiters := []struct {
		name    string
		limiter Limiter
	}{
		{"TokenBucket", NewTokenBucketLimiter(rdb, 5, 1)},
		{"FixedWindow", NewFixedWindowLimiter(rdb, 5, time.Minute)},
		{"SlidingWindow", NewSlidingWindowLimiter(rdb, 5, time.Minute)},
	}

	ctx := context.Background()
	for _, l := range limiters {
		allowed, err := l.limiter.Allow(ctx, "interface-user")
		if err != nil {
			t.Fatalf("%s: %v", l.name, err)
		}
		fmt.Printf("%s interface test: allowed=%v ✓\n", l.name, allowed)
	}
}