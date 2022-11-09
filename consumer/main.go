package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v9"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"net"
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

	go func() {
		data, err := ensureSocketIsEmpty(logger)
		if err != nil {
			logger.Error("failed to read from socket: ", err)
		}
		if data == nil {
			logger.Info("no sockets available")
		}
	}()

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

// ensureSocketIsEmpty trying to read data from socket every 0.2 seconds
// systemd relaunches the service if the data from socket has not read
func ensureSocketIsEmpty(logger *zap.SugaredLogger) ([]byte, error) {
	sock, err := utils.GetSystemdSocket()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get systemd socket")
	}
	if sock == nil {
		// no sockets available
		return nil, nil
	}
	defer sock.Close()

	for {
		data, err := readFromSocket(sock)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read from socket")
		}
		if data != nil {
			logger.Debugf("data read from socket: %s", string(data))
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func readFromSocket(sock net.Listener) ([]byte, error) {
	conn, err := sock.Accept()
	if err != nil {
		return nil, errors.Wrap(err, "failed to accept connection")
	}
	defer conn.Close()

	data := make([]byte, 1024)
	for {
		buf := make([]byte, 1024) // 1Kb buffer should be enough for our purposes
		_, err = conn.Read(buf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		} else if err != nil {
			return nil, errors.Wrap(err, "failed to read from socket")
		}
		data = append(data, buf...)
	}
	return data, nil
}
