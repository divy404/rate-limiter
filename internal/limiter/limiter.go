package limiter

import "context"

// Any struct with an Allow method automatically satisfies this interface
type Limiter interface {
	Allow(ctx context.Context, clientID string) (bool, error)
}

const (
	StrategyTokenBucket   = "token_bucket"
	StrategyFixedWindow   = "fixed_window"
	StrategySlidingWindow = "sliding_window"
)