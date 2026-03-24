package queue

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
)

type RechargeEvent struct {
	MSISDN string `json:"msisdn"`
	Amount int64  `json:"amount"`
	Ref    string `json:"ref"`
}

type EventQueue struct {
	client *redis.Client
	stream string
}

func NewEventQueue(client *redis.Client, stream string) *EventQueue {
	return &EventQueue{client: client, stream: stream}
}

func (q *EventQueue) PushRecharge(ctx context.Context, event RechargeEvent) error {
	data, _ := json.Marshal(event)
	return q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.stream,
		Values: map[string]interface{}{"payload": data},
	}).Err()
}
