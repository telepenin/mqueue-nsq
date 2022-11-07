package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v9"
	"go.uber.org/zap"
	"os"
	"strconv"
	"time"

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

	var timeout int
	timeoutS := os.Getenv("TIMEOUT")
	if timeoutS != "" {
		value, err := strconv.Atoi(timeoutS)
		if err != nil {
			logger.Error("failed to parse TIMEOUT: ", err)
			return
		}
		timeout = value
	} else {
		timeout = 0
	}

	ctx := context.TODO()
	event := api.NewEvent(client)
	err := event.CreateGroup(ctx, stream, group)
	if err != nil && !api.GroupAlreadyExists(err) {
		logger.Error("failed to create group: ", err)
		return
	}

	for {
		messages, err := event.ReadByGroup(ctx, stream, group, time.Duration(timeout)*time.Second)
		if err != nil {
			if api.TimeoutExceeded(err) {
				logger.Infof("waiting new messages timeout %ds exceeded", timeout)
				return
			}
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
