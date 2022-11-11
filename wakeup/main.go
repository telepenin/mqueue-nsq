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
	streams, err := mapStream(ctx, event)
	if err != nil {
		logger.Error("failed to list streams: ", err)
		return
	}
	logger.Debugw("streams was found", "streams", streams)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// subscribe to new messages in all streams
			logger.Infow("subscribing to new messages", "streams", listStream(streams))

			result, err := event.Read(ctx, listStream(streams), 0)
			if err != nil {
				logger.Error("failed to read from redis: ", err)
				return
			}
			for _, v := range result {
				groups, ok := streams[v.Stream]
				if ok {
					for _, group := range groups {
						if err := wakeup(group.Name); err != nil {
							logger.Error("failed to wakeup: ", err)
							continue
						}
						logger.Infow("wakeup is done", "stream", v.Stream, "addr", group.Name)
					}
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("checking unprocessed messages")
		for _, stream := range streams {
			for _, group := range stream {
				logger.Infow("checking group", "stream", stream, "group", group)
				if group.HasUnprocessed {
					logger.Infow("initial wakeup for group is starting...", "stream", stream, "addr", group.Name)
					if err := wakeup(group.Name); err != nil {
						logger.Error("failed to wakeup: ", err)
						return
					}
					logger.Infow("initial wakeup is done", "stream", stream, "addr", group.Name)
				}
			}
		}
		logger.Info("initial wakeup is done for all groups")
	}()

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

// listStreams returns map of stream names and their groups.
//(group - (name: addr of socket, HasUnprocessed - indicates should we wake up for reading old messages or not) , stream).
// it collects redis streams and their groups
// checks the group name as a valid socket file
func mapStream(ctx context.Context, event *api.Event) (map[string][]api.Group, error) {
	streams, err := event.ListStream(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list streams")
	}
	result := make(map[string][]api.Group)
	for _, stream := range streams {
		groups, err := event.ListGroup(ctx, stream)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list groups for %s", stream)
		}
		socketGroups := make([]api.Group, 0, len(groups))
		for _, group := range groups {
			if utils.IsSocket(group.Name) {
				socketGroups = append(socketGroups, group)
			}
		}
		if len(socketGroups) > 0 {
			result[stream] = socketGroups
		}
	}
	return result, nil
}

func listStream(m map[string][]api.Group) []string {
	var result []string
	for k := range m {
		result = append(result, k)
	}
	return result
}
