package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client *redis.Client
	limit int
	window time.Duration
}

type Result struct {
	Allowed bool
	Remaining int
	Reset time.Duration	
}

func New(client *redis.Client, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		client: client,
		limit: limit,
		window: window,
	}
}

func (rl *RateLimiter) Allow(ctx context.Context, identifier string) (*Result,error) {
	key := fmt.Sprintf("ratelimit:%s",identifier)
	now := time.Now()
	nowNano := now.UnixNano()
	clearBefore := now.Add(-rl.window).UnixNano()
	// Atomic pipeline to prevent multi-pod race conditions
	pipe := rl.client.TxPipeline()
	// 1. Remove old requests outside current window
	pipe.ZRemRangeByScore(ctx,key,"0",fmt.Sprintf("%d",clearBefore))
	
	// 2. Add current request attempt tentatively
	pipe.ZAdd(ctx,key,redis.Z{
		Score: float64(nowNano),
		Member: fmt.Sprintf("%d",nowNano),
	})
	// 3. Count requests currently in the window
	cardCmd := pipe.ZCard(ctx,key)
	// 4. Refresh TTL so idle keys expire automatically
	pipe.Expire(ctx,key,rl.window+time.Second)
	// Execute pipeline
	_,err := pipe.Exec(ctx)
	if err != nil {
		return nil,fmt.Errorf("failed to execute rate loimit pipeline: %w", err)
	}
	totalInWindow := int(cardCmd.Val())
	if totalInWindow > rl.limit {
		_= rl.client.ZRem(ctx, key, fmt.Sprintf("%d", nowNano)).Err()
		return &Result{
			Allowed: false,
			Remaining: 0,
			Reset: rl.window,
		}, nil
	}
	return &Result{
		Allowed: true,
		Remaining: rl.limit - totalInWindow,
		Reset: rl.window,
	},nil
}