// Package queue provides a Redis-backed job queue for background task processing.
// When REDIS_URL is not configured or the connection is unavailable, all
// queue operations are no-ops so the API server runs cleanly without Redis.
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// EventQueue is a Redis Streams-based event queue for async processing.
type EventQueue struct {
	rdb    *redis.Client
	stream string
}

func NewEventQueue(rdb *redis.Client, stream string) *EventQueue {
	return &EventQueue{rdb: rdb, stream: stream}
}

// isAvailable pings Redis and returns false when unavailable (no URL set,
// connection refused, EOF on Render free tier, etc.).  This prevents the
// Subscribe goroutine from logging an error on every 32-second retry cycle.
func (q *EventQueue) isAvailable(ctx context.Context) bool {
	if q.rdb == nil {
		return false
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return q.rdb.Ping(pingCtx).Err() == nil
}

func (q *EventQueue) Publish(ctx context.Context, event map[string]interface{}) error {
	if !q.isAvailable(ctx) {
		return nil // no-op when Redis unavailable
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return q.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: q.stream,
		Values: map[string]interface{}{
			"event": event["type"],
			"data":  string(data),
			"ts":    time.Now().Unix(),
		},
	}).Err()
}

func (q *EventQueue) Subscribe(ctx context.Context, group, consumer string, handler func(map[string]interface{}) error) {
	// Check availability before starting the loop. If Redis is not configured
	// (Render free tier, local dev without Redis), skip silently — the
	// leaderboard WebSocket poller provides equivalent real-time updates.
	if !q.isAvailable(ctx) {
		log.Printf("[QUEUE] Redis unavailable — leaderboard stream disabled (WebSocket poller active)")
		return
	}

	// Create consumer group if it doesn't exist
	q.rdb.XGroupCreateMkStream(ctx, q.stream, group, "0")

	backoff := 2 * time.Second
	const maxBackoff = 64 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		entries, err := q.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    group,
			Consumer: consumer,
			Streams:  []string{q.stream, ">"},
			Count:    10,
			Block:    5 * time.Second,
		}).Result()

		if err == redis.Nil || err == context.DeadlineExceeded {
			backoff = 2 * time.Second
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			// Re-check availability; if Redis dropped, exit the loop and let
			// the caller decide whether to restart (avoids log flooding).
			if !q.isAvailable(ctx) {
				log.Printf("[QUEUE] Redis connection lost — leaderboard stream paused")
				return
			}
			log.Printf("[QUEUE] XReadGroup error (retrying in %s): %v", backoff, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < maxBackoff {
				backoff *= 2
			}
			continue
		}

		backoff = 2 * time.Second

		for _, stream := range entries {
			for _, msg := range stream.Messages {
				var event map[string]interface{}
				if dataStr, ok := msg.Values["data"].(string); ok {
					if unmarshalErr := json.Unmarshal([]byte(dataStr), &event); unmarshalErr != nil {
						log.Printf("[QUEUE] unmarshal error for msg %s: %v", msg.ID, unmarshalErr)
					}
				}
				if err := handler(event); err != nil {
					log.Printf("[QUEUE] handler error for %s: %v", msg.ID, err)
					continue
				}
				q.rdb.XAck(ctx, q.stream, group, msg.ID)
			}
		}
	}
}

func (q *EventQueue) Length(ctx context.Context) (int64, error) {
	if !q.isAvailable(ctx) {
		return 0, nil
	}
	return q.rdb.XLen(ctx, q.stream).Result()
}

func (q *EventQueue) StreamName() string {
	return fmt.Sprintf("redis-stream:%s", q.stream)
}
