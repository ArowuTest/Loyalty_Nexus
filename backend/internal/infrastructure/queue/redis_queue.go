// Package queue provides a Redis-backed job queue for background task processing.
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

func (q *EventQueue) Publish(ctx context.Context, event map[string]interface{}) error {
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
	// Create consumer group if it doesn't exist
	q.rdb.XGroupCreateMkStream(ctx, q.stream, group, "0")

	// Exponential backoff on connection errors to avoid log flooding.
	// Render's free-tier Redis resets idle TCP connections frequently;
	// the go-redis client reconnects automatically on the next call,
	// so we just need to wait briefly before retrying.
	backoff := time.Second
	const maxBackoff = 30 * time.Second

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
			// Normal: no messages / block timeout — reset backoff and continue
			backoff = time.Second
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				// Context cancelled — clean shutdown, not an error
				return
			}
			// Connection error (EOF, reset, etc.) — log once per backoff window,
			// then wait before retrying so we don't flood the log.
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

		// Successful read — reset backoff
		backoff = time.Second

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
					// NAK — will be retried
					continue
				}
				// ACK on success
				q.rdb.XAck(ctx, q.stream, group, msg.ID)
			}
		}
	}
}

func (q *EventQueue) Length(ctx context.Context) (int64, error) {
	return q.rdb.XLen(ctx, q.stream).Result()
}

func (q *EventQueue) StreamName() string {
	return fmt.Sprintf("redis-stream:%s", q.stream)
}
