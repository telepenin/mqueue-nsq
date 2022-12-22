package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"net"
	"os"
	"os/signal"
	"path"
	"strconv"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/nsqio/go-nsq"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"mqueue/pkg/utils"
)

type handler struct {
	stream string
	group  string
	logger *zap.SugaredLogger

	reset      func()
	timestamps []int64 // checks inordering for messages
}

var _ nsq.Handler = &handler{}

func main() {
	zapLogger, _ := zap.NewDevelopment()
	defer zapLogger.Sync()
	logger := zapLogger.Sugar()

	logger.Info("consumer is starting...")

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

	var group string
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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sock, err := utils.GetSystemdSocket()
	if err != nil {
		logger.Error("failed to get systemd socket: ", err)
		return
	}
	if sock == nil {
		logger.Info("no sockets available")
		group = os.Getenv("GROUP")
		if group == "" {
			logger.Error("GROUP env var is not set\n")
			return
		}
	} else {
		defer sock.Close()
		group = path.Base(sock.Addr().String()) // use basename of group as socket filename

		go func() {
			if err = ensureSocketIsDisconnected(sock); err != nil {
				logger.Error("failed to ensure socket is disconnected: ", err)
			}
		}()
	}

	logger.Debugf("group: %s, stream: %s, timeout: %ds", group, stream, timeout)

	cfg := nsq.NewConfig()
	//cfg.MaxAttempts = 2
	//cfg.DefaultRequeueDelay = 5 * time.Second
	//cfg.BackoffMultiplier = 0

	consumer, err := nsq.NewConsumer(stream, group, cfg)
	if err != nil {
		logger.Error("failed to create consumer: ", err)
		return
	}
	ctx := context.Background()
	ctx, _, reset := WithTimeoutReset(ctx, time.Duration(timeout)*time.Second)

	h := &handler{
		stream:     stream,
		group:      group,
		logger:     logger,
		reset:      reset,
		timestamps: make([]int64, 0),
	}
	consumer.AddHandler(h)

	// creates implicitly topic & channel
	if err = consumer.ConnectToNSQD(addr); err != nil {
		logger.Error("failed to connect to nsqd: ", err)
		return
	}

	select {
	case <-sigChan:
	case <-ctx.Done():
	}
	logger.Info("consumer is stopping...")

	// check timestamps
	isOrdered := true
	for i := 0; i < len(h.timestamps); i++ {
		if i == len(h.timestamps)-1 {
			break
		}
		//logger.Debugw("timestamps: ",
		//	"i", i,
		//	"ts", h.timestamps[i],
		//	"next", h.timestamps[i+1],
		//	"since", time.Since(time.Unix(0, h.timestamps[i])))
		if h.timestamps[i] > h.timestamps[i+1] {
			isOrdered = false
			logger.Errorf("message %d is out of order", i)
		}
	}

	logger.Infof("messages are ordered: %t", isOrdered)

	consumer.Stop()
	<-consumer.StopChan
}

func (h *handler) HandleMessage(message *nsq.Message) error {
	defer h.reset() // reset the awaiting timeout

	var data interface{}
	err := json.Unmarshal(message.Body, &data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal event")
	}

	h.logger.Infow("event read: ",
		"id", string(message.ID[:]),
		"size", humanize.Bytes(uint64(binary.Size(message.Body))),
		"stream", h.stream,
		"group", h.group,
		"data", data,
	)

	h.timestamps = append(h.timestamps, message.Timestamp)
	// ack by default if returned nil
	//return errors.New("failed to process event")
	return nil
}

func WithTimeoutReset(parent context.Context, d time.Duration) (context.Context, context.CancelFunc, func()) {
	ctx, cancel0 := context.WithCancel(parent)
	timer := time.AfterFunc(d, cancel0)
	cancel := func() {
		cancel0()
		timer.Stop()
	}
	reset := func() { timer.Reset(d) }
	return ctx, cancel, reset
}

// ensureSocketIsDisconnected trying to accept/close connect from socket every 0.2 seconds
// systemd relaunches the service if it has open connection
func ensureSocketIsDisconnected(sock net.Listener) error {
	for {
		conn, err := sock.Accept()
		if err != nil {
			return errors.Wrap(err, "failed to accept connection")
		}
		if err := conn.Close(); err != nil {
			return errors.Wrap(err, "failed to close connection")
		}
		time.Sleep(200 * time.Millisecond)
	}
}
