package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/nsqio/go-nsq"
	"github.com/thanhpk/randstr"
	"github.com/tjarratt/babble"
	"go.uber.org/zap"

	"mqueue/pkg/event"
)

func main() {
	zapLogger, _ := zap.NewDevelopment()
	defer zapLogger.Sync()
	logger := zapLogger.Sugar()

	logger.Info("producer is starting...")

	addr := os.Getenv("NSQ_ADDR")
	if addr == "" {
		logger.Error("NSQ_ADDR env var is not set\n")
		return
	}

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
	//ctx := context.TODO()

	producer, err := nsq.NewProducer(addr, nsq.NewConfig())
	if err != nil {
		logger.Error("failed to create producer: ", err)
		return
	}

	stopChan := make(chan bool)
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	for i := 0; i < goroutines; i++ {
		logger.Debugf("starting goroutine %d", i)

		go func() {
			ticker := time.NewTicker(timeout)
			for {
				e := event.Event{
					Payload: makePayload(payloadKeys),
				}
				bytes, err := e.MarshalBinary()
				if err != nil {
					logger.Error("failed to marshal event: ", err)
					close(stopChan)
					return
				}

				select {
				case <-ticker.C:
					err := producer.Publish(stream, bytes)
					if err != nil {
						logger.Error("failed to create event: ", err)
						close(stopChan)
						return
					}
					logger.Infow("event created: ", "id", "-", "stream", stream,
						"size", humanize.Bytes(uint64(binary.Size(bytes))))
				}
			}
		}()
	}

	select {
	case <-termChan:
	case <-stopChan:
	}

	logger.Info("producer is stopping...")
	producer.Stop()
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
