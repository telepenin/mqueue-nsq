package main

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/go-redis/redis/v9"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"mqueue/pkg/api"
	"mqueue/pkg/utils"
)

var (
	client *redis.Client
)

type consumer struct {
	Name   string
	Stream string
}

func init() {
	var err error
	client, err = utils.NewRedisClient()
	if err != nil {
		panic(fmt.Sprintf("failed to connect to redis: %v", err))
	}
}

func main() {
	zapLogger, _ := zap.NewDevelopment()
	defer zapLogger.Sync()
	logger := zapLogger.Sugar()

	logger.Info("wakeup service is starting...")
	event := api.NewEvent(client)

	ctx := context.TODO()
	consumers, err := listConsumers(ctx, event)
	if err != nil {
		logger.Error("failed to list consumers: ", err)
		return
	}
	if len(consumers) == 0 {
		logger.Info("no consumers found")
		return
	}
	logger.Debugw("consumers is found", "consumers", consumers)

	wg := sync.WaitGroup{}
	group := "wakeup"
	for _, c := range consumers {
		// creating wakeup group for each stream
		err := event.CreateGroup(ctx, c.Stream, group)
		if err != nil && !api.GroupAlreadyExists(err) {
			logger.Error("failed to create group: ", err)
			return
		}

		wg.Add(1)

		go func(stream, addr string) {
			defer wg.Done()
			logger.Infow("wakeup for stream is starting...", "stream", stream, "addr", addr)
			for {
				logger.Infow("waiting for events", "stream", stream, "addr", addr)
				messages, err := event.ReadByGroup(ctx, stream, group, 0)
				if err != nil {
					logger.Error("failed to read event: ", err)
					return
				}
				logger.Debugln("messages: ", len(messages))
				if len(messages) > 0 {
					logger.Infow("Wakeup! New messages is arriving", "stream", stream, "addr", addr)
					if err := wakeup(addr); err != nil {
						logger.Error("failed to wakeup: ", err)
						continue
					}
					logger.Infow("wakeup is done", "stream", stream, "addr", addr)
				}
				// ack messages
				for _, v := range messages {
					logger.Infow("event read: ", "id", v.ID, "payload", v.Values, "stream", stream)
					err = event.AckByGroup(ctx, stream, group, v.ID)
					if err != nil {
						logger.Error("failed to ack event: ", err)
						return
					}
				}
			}
		}(c.Stream, c.Name)
	}
	wg.Wait()
}

// wakeup open connection to the given socket address.
func wakeup(addr string) error {
	conn, err := net.Dial("unix", addr)
	if err != nil {
		return errors.Wrap(err, "failed to connect to unix socket")
	}
	defer conn.Close()
	return nil
}

// listConsumers returns list of consumers (name - addr of socket, stream).
// it collects redis streams and their groups
// checks the group name as a valid socket file
func listConsumers(ctx context.Context, event *api.Event) ([]consumer, error) {
	streams, err := event.ListStream(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list streams")
	}
	consumers := make([]consumer, 0)
	for _, stream := range streams {
		groups, err := event.ListGroup(ctx, stream)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list groups for %s", stream)
		}
		for _, name := range groups {
			if utils.IsSocket(name) {
				consumers = append(consumers, consumer{
					Name:   name,
					Stream: stream,
				})
			}
		}
	}
	return consumers, nil
}
