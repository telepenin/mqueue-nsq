package api

import (
	"context"

	"github.com/go-redis/redis/v9"

	"mqueue/pkg/event"
)

type Event struct {
	client *redis.Client
}

func NewEvent(client *redis.Client) *Event {
	return &Event{client: client}
}

func (e *Event) Create(ctx context.Context, stream string, message *event.Event) (string, error) {
	return e.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: message.Payload,
	}).Result()
}

func (e *Event) ReadByGroup(ctx context.Context, stream string, group string) ([]redis.XMessage, error) {
	val, err := e.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Streams:  []string{stream, ">"},
		Group:    group,
		Consumer: "consumer",
		Count:    10,
		Block:    0,
	}).Result()
	if err != nil {
		return nil, err
	}
	// 0 is the index of the requested stream
	return val[0].Messages, nil
}

func (e *Event) AckByGroup(ctx context.Context, stream string, group string, id string) error {
	return e.client.XAck(ctx, stream, group, id).Err()
}

func (e *Event) CreateGroup(ctx context.Context, stream string, group string) error {
	return e.client.XGroupCreate(ctx, stream, group, "$").Err()
}

func GroupAlreadyExists(err error) bool {
	return err.Error() == "BUSYGROUP Consumer Group name already exists"
}
