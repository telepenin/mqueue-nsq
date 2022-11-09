package api

import (
	"context"
	"github.com/go-redis/redis/v9"
	"time"

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

func (e *Event) ReadByGroup(ctx context.Context, stream string, group string, timeout time.Duration) ([]redis.XMessage, error) {
	val, err := e.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Streams:  []string{stream, ">"},
		Group:    group,
		Consumer: "consumer",
		Count:    10,
		Block:    timeout,
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
	return e.client.XGroupCreateMkStream(ctx, stream, group, "$").Err()
}

func (e *Event) ListStream(ctx context.Context) ([]string, error) {
	keys, _, err := e.client.ScanType(ctx, 0, "", 100, "stream").Result()
	return keys, err
}

func (e *Event) ListGroup(ctx context.Context, stream string) ([]string, error) {
	result, err := e.client.XInfoGroups(ctx, stream).Result()
	if err != nil {
		return nil, err
	}
	groups := make([]string, 0, len(result))
	for _, v := range result {
		groups = append(groups, v.Name)
	}
	return groups, nil
}

func GroupAlreadyExists(err error) bool {
	return err.Error() == "BUSYGROUP Consumer Group name already exists"
}

func TimeoutExceeded(err error) bool {
	return err.Error() == "redis: nil"
}
