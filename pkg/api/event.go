package api

import (
	"context"
	"github.com/pkg/errors"
	"time"

	"github.com/go-redis/redis/v9"

	"mqueue/pkg/event"
)

type Event struct {
	client *redis.Client
}

type Group struct {
	Name           string
	HasUnprocessed bool
}

func NewEvent(client *redis.Client) *Event {
	return &Event{client: client}
}

// Create message using stream
func (e *Event) Create(ctx context.Context, stream string, message *event.Event) (string, error) {
	return e.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: message.Payload,
	}).Result()
}

// Read new messages in streams
func (e *Event) Read(ctx context.Context, streams []string, timeout time.Duration) ([]redis.XStream, error) {
	args := streams
	for range streams {
		args = append(args, "$") // $ means new messages
	}
	val, err := e.client.XRead(ctx, &redis.XReadArgs{
		Streams: args,
		Count:   10,
		Block:   timeout,
	}).Result()
	if err != nil {
		return nil, err
	}
	return val, nil
}

// ReadByGroup reads messages from stream by specific group
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

// AckByGroup acks messages by group
func (e *Event) AckByGroup(ctx context.Context, stream string, group string, id string) error {
	return e.client.XAck(ctx, stream, group, id).Err()
}

// CreateGroup creates group for stream
func (e *Event) CreateGroup(ctx context.Context, stream string, group string) error {
	return e.client.XGroupCreateMkStream(ctx, stream, group, "$").Err()
}

// ListStream returns list of streams
func (e *Event) ListStream(ctx context.Context) ([]string, error) {
	keys, _, err := e.client.ScanType(ctx, 0, "", 100, "stream").Result()
	return keys, err
}

// ListGroup returns list of groups for stream
func (e *Event) ListGroup(ctx context.Context, stream string) ([]Group, error) {
	result, err := e.client.XInfoGroups(ctx, stream).Result()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get groups info")
	}
	groups := make([]Group, len(result))
	for _, v := range result {
		groups = append(groups, Group{
			Name:           v.Name,
			HasUnprocessed: v.Pending > 0 || v.Lag > 0,
		})
	}
	return groups, nil
}

func (e *Event) Trim(ctx context.Context, stream string, count int64) error {
	return e.client.XTrimMaxLenApprox(ctx, stream, count, 0).Err()
}

// GroupAlreadyExists returns true if group already exists
func GroupAlreadyExists(err error) bool {
	return err.Error() == "BUSYGROUP Consumer Group name already exists"
}

// TimeoutExceeded returns true if timeout exceeded
func TimeoutExceeded(err error) bool {
	return err.Error() == "redis: nil"
}
