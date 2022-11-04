package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-redis/redis/v9"
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

	stream := os.Getenv("STREAM")
	if stream == "" {
		logger.Error("STREAM env var is not set\n")
		return
	}

	group := os.Getenv("GROUP")
	if group == "" {
		logger.Error("GROUP env var is not set\n")
		return
	}
	ctx := context.TODO()

	event := api.NewEvent(client)
	err := event.CreateGroup(ctx, stream, group)
	if err != nil && !api.GroupAlreadyExists(err) {
		logger.Error("failed to create group: ", err)
		return
	}

	for {
		messages, err := event.ReadByGroup(ctx, stream, group)
		if err != nil {
			logger.Error("failed to read event: ", err)
			return
		}
		logger.Debugln("messages: ", len(messages))
		for _, v := range messages {
			logger.Infow("event read: ", "id", v.ID, "payload", v.Values, "stream", stream)

			err = event.AckByGroup(ctx, stream, group, v.ID)
			if err != nil {
				logger.Error("failed to ack event: ", err)
				return
			}
		}
	}
}
