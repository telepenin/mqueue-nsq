package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
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

	logger.Info("consumer is starting...")

	targets := os.Getenv("TARGETS")
	if targets == "" {
		logger.Error("TARGETS env var is not set\n")
		return
	}

	wg := sync.WaitGroup{}
	group := "wakeup"
	ctx := context.TODO()
	event := api.NewEvent(client)
	for _, value := range strings.Split(targets, ",") {
		v := strings.SplitN(value, ":", 2)
		stream, sockAddr := v[0], v[1]

		err := event.CreateGroup(ctx, stream, group)
		if err != nil && !api.GroupAlreadyExists(err) {
			logger.Error("failed to create group: ", err)
			return
		}

		wg.Add(1)
		go func(stream string) {
			defer wg.Done()
			logger.Infow("wakeup for stream is starting...", "stream", stream, "sockAddr", sockAddr)
			for {
				logger.Infow("waiting for events", "stream", stream, "sockAddr", sockAddr)
				messages, err := event.ReadByGroup(ctx, stream, group, 0)
				if err != nil {
					logger.Error("failed to read event: ", err)
					return
				}
				logger.Debugln("messages: ", len(messages))
				if len(messages) > 0 {
					logger.Infow("Hei! Hei! Hei! Wakeup! New messages is arriving", "stream", stream, "sockAddr", sockAddr)
					if err := wakeup(sockAddr); err != nil {
						logger.Error("failed to wakeup: ", err)
						continue
					}
					logger.Infow("wakeup is done", "stream", stream, "sockAddr", sockAddr)
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
		}(stream)
	}
	wg.Wait()
}

func wakeup(sockAddr string) error {
	conn, err := net.Dial("unix", sockAddr)
	if err != nil {
		return errors.Wrap(err, "failed to connect to unix socket")
	}
	defer conn.Close()
	_, err = conn.Write([]byte("WAKEUP"))
	if err != nil {
		return errors.Wrap(err, "failed to write to unix socket")
	}
	return nil
}
