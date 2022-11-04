package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/tjarratt/babble"
	"go.uber.org/zap"

	"mqueue/pkg/api"
	"mqueue/pkg/event"
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

	logger.Info("producer is starting...")

	stream := os.Getenv("STREAM")
	if stream == "" {
		logger.Error("STREAM env var is not set\n")
		return
	}

	ev := api.NewEvent(client)

	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			ctx := context.TODO()
			payload := map[string]interface{}{
				"message": babble.NewBabbler().Babble(),
			}
			val, err := ev.Create(ctx, stream, &event.Event{
				Payload: payload,
			})
			if err != nil {
				logger.Error("failed to create event: ", err)
				return
			}
			logger.Infow("event created: ", "id", val, "payload", payload, "stream", stream)
		}
	}
}
