// Package queue provides a Redis-backed job queue for background task processing.
// When REDIS_URL is not configured or the connection fails, operations degrade
// gracefully: Publish is a no-op and Subscribe exits cleanly.
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
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

// isTransientConnErr returns true for connection-level errors that go-redis
// will recover from automatically on the next call (EOF, connection reset,
// broken pipe, TLS close_notify). These are expected on Render / any cloud
// Redis that kills idle TLS connections and should NOT be logged as errors.
func isTransientConnErr(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "eof") ||
		strings.Contains(s, "connection reset") ||
		strings.Contains(s, "broken pipe") ||
		strings.Contains(s, "use of closed") ||
		strings.Contains(s, "tls") ||
		strings.Contains(s, "i/o timeout")
}

// isAvailable does a quick ping to check whether Redis is reachable at all.
// Used only at startup to decide whether to print a "disabled" notice.
func (q *EventQueue) isAvailable(ctx context.Context) bool {
	if q.rdb == nil {
		return false
	}
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return q.rdb.Ping(pingCtx).Err() == nil
}

func (q *EventQueue) Publish(ctx context.Context, event map[string]interface{}) error {
	if q.rdb == nil {
		return nil
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
	// Check once at startup — if Redis is completely unreachable, skip silently.
	if !q.isAvailable(ctx) {
		log.Printf("[QUEUE] Redis not reachable at startup — leaderboard stream disabled")
		return
	}

	// Create consumer group (idempotent — OK if already exists)
	_ = q.rdb.XGroupCreateMkStream(ctx, q.stream, group, "0")
	log.Printf("[QUEUE] Subscribed to stream=%s group=%s consumer=%s", q.stream, group, consumer)

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
			// Normal: block timeout, no messages — continue immediately
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return // clean shutdown
			}
			// Transient connection error (EOF, TLS reset, etc.): Render's Redis
			// proxy kills idle connections periodically. go-redis reconnects on
			// the next XReadGroup call automatically — just wait a moment and retry.
			if isTransientConnErr(err) {
				select {
				case <-ctx.Done():
					return
				case <-time.After(500 * time.Millisecond):
				}
				continue
			}
			// Non-transient error (WRONGTYPE, NOAUTH, etc.) — log and back off
			log.Printf("[QUEUE] XReadGroup unexpected error (will retry): %v", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}

		// Successful read — process messages
		for _, stream := range entries {
			for _, msg := range stream.Messages {
				var event map[string]interface{}
				if dataStr, ok := msg.Values["data"].(string); ok {
					_ = json.Unmarshal([]byte(dataStr), &event)
				}
				if handlerErr := handler(event); handlerErr != nil {
					log.Printf("[QUEUE] handler error for msg %s: %v", msg.ID, handlerErr)
					continue // NAK — retry on next poll
				}
				_ = q.rdb.XAck(ctx, q.stream, group, msg.ID)
			}
		}
	}
}

func (q *EventQueue) Length(ctx context.Context) (int64, error) {
	if q.rdb == nil {
		return 0, nil
	}
	return q.rdb.XLen(ctx, q.stream).Result()
}

func (q *EventQueue) StreamName() string {
	return fmt.Sprintf("redis-stream:%s", q.stream)
}
