package external

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisUsageTracker struct {
	rdb *redis.Client
}

func NewRedisUsageTracker(rdb *redis.Client) *RedisUsageTracker {
	return &RedisUsageTracker{rdb: rdb}
}

func (t *RedisUsageTracker) GetDailyCount(ctx context.Context, userID string) (int, error) {
	key := dailyKey(userID)
	val, err := t.rdb.Get(ctx, key).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

func (t *RedisUsageTracker) Increment(ctx context.Context, userID string) error {
	key := dailyKey(userID)
	pipe := t.rdb.Pipeline()
	pipe.Incr(ctx, key)
	pipe.ExpireAt(ctx, key, endOfDay())
	_, err := pipe.Exec(ctx)
	return err
}

func dailyKey(userID string) string {
	return fmt.Sprintf("chat:daily:%s:%s", userID, time.Now().Format("2006-01-02"))
}

func endOfDay() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
}
