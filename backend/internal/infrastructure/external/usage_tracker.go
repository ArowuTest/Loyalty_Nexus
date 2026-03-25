package external

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

type RedisUsageTracker struct {
	rdb *redis.Client
}

func NewRedisUsageTracker(rdb *redis.Client) *RedisUsageTracker {
	return &RedisUsageTracker{rdb: rdb}
}

func (t *RedisUsageTracker) GetDailyCount(ctx context.Context, userID string) (int, error) {
	key := fmt.Sprintf("usage:llm:%s:%s", userID, time.Now().Format("2006-01-02"))
	val, err := t.rdb.Get(ctx, key).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

func (t *RedisUsageTracker) Increment(ctx context.Context, userID string) error {
	key := fmt.Sprintf("usage:llm:%s:%s", userID, time.Now().Format("2006-01-02"))
	err := t.rdb.Incr(ctx, key).Err()
	if err == nil {
		t.rdb.Expire(ctx, key, 24*time.Hour)
	}
	return err
}
