package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/DmitriyVTitov/size"
	"github.com/go-redis/redis/v9"
	"github.com/thanhpk/randstr"
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

	goroutines := 1
	if os.Getenv("GOROUTINES") != "" {
		v, err := strconv.Atoi(os.Getenv("GOROUTINES"))
		if err != nil {
			logger.Error("failed to parse GOROUTINES: ", err)
			return
		}
		goroutines = v
	}

	payloadKeys := 1
	if os.Getenv("PAYLOAD_KEYS") != "" {
		v, err := strconv.Atoi(os.Getenv("PAYLOAD_KEYS"))
		if err != nil {
			logger.Error("failed to parse PAYLOAD_KEYS: ", err)
			return
		}
		payloadKeys = v
	}

	timeout := 1 * time.Second
	if os.Getenv("TIMEOUT") != "" {
		value, err := strconv.Atoi(os.Getenv("TIMEOUT"))
		if err != nil {
			logger.Error("failed to parse TIMEOUT: ", err)
			return
		}
		timeout = time.Duration(value) * time.Millisecond
	}

	ev := api.NewEvent(client)
	wg := sync.WaitGroup{}
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		logger.Debugf("starting goroutine %d", i)

		go func() {
			defer wg.Done()

			ticker := time.NewTicker(timeout)
			for {
				payload := makePayload(payloadKeys)
				select {
				case <-ticker.C:
					val, err := ev.Create(context.TODO(), stream, &event.Event{
						Payload: payload,
					})
					if err != nil {
						logger.Error("failed to create event: ", err)
						return
					}
					logger.Infow("event created: ", "id", val, "stream", stream,
						"size", size.Of(payload))
				}
			}
		}()
	}
	wg.Wait()
}

func makePayload(keys int) map[string]interface{} {
	payload := make(map[string]interface{}, keys)
	for i := 0; i < keys; i++ {
		var data string
		if os.Getenv("USE_WORDS") != "" {
			data = babble.NewBabbler().Babble()
		} else {
			data = randstr.Hex(16)
		}
		payload[fmt.Sprintf("key_%d", i)] = data
	}
	return payload
}
