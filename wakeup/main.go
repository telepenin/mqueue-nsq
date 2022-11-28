package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/nsqio/go-nsq"
	"github.com/nsqio/nsq/nsqd"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"mqueue/pkg/api"
	"mqueue/pkg/utils"
)

type handler struct {
	stream string
	logger *zap.SugaredLogger
	groups []api.Group
}

func main() {
	zapLogger, _ := zap.NewDevelopment()
	defer zapLogger.Sync()
	logger := zapLogger.Sugar()

	logger.Info("wakeup service is starting...")

	addr := os.Getenv("NSQ_ADDR")
	if addr == "" {
		logger.Error("NSQ_ADDR env var is not set\n")
		return
	}

	httpAddr := os.Getenv("NSQ_ADDR_HTTP")
	if httpAddr == "" {
		logger.Error("NSQ_ADDR_HTTP env var is not set\n")
		return
	}

	topics, err := getTopics(httpAddr)
	if err != nil {
		logger.Error("failed to get topics: ", err)
		return
	}

	listeners := activeListeners(topics)
	logger.Infow("listeners", "listeners", listeners)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// wakeup all active listeners if there have unprocessed messages
	go func() {
		logger.Infow("pre wakeup services")
		for topic, groups := range listeners {
			for _, group := range groups {
				if group.HasUnprocessed {
					logger.Infow("waking up", "topic", topic, "group", group.Name)
					if err := wakeup(path.Join("/run", group.Name)); err != nil {
						logger.Errorw("failed to wake up", "topic", topic, "group", group.Name, "error", err)
					}
				}
			}
		}
	}()

	var consumers []*nsq.Consumer
	// subscribe to all topics with active listeners
	for topic, groups := range listeners {
		// create consumer for each topic
		logger.Infow("subscribing", "topic", topic)

		consumer, err := nsq.NewConsumer(topic, "wakeup", nsq.NewConfig())
		if err != nil {
			logger.Error("failed to create consumer: ", err)
			return
		}
		h := &handler{
			stream: topic,
			logger: logger,
			groups: groups,
		}
		consumer.AddHandler(h)

		if err := consumer.ConnectToNSQD(addr); err != nil {
			logger.Error("failed to connect to nsqd: ", err)
			return
		}

		consumers = append(consumers, consumer)
	}

	<-sigChan
	logger.Info("wakeup service is shutting down...")

	for _, consumer := range consumers {
		consumer.Stop()
		<-consumer.StopChan
	}
}

func (h *handler) HandleMessage(_ *nsq.Message) error {
	h.logger.Infow("received message", "stream", h.stream)

	for _, group := range h.groups {
		h.logger.Info("waking up", "group", group.Name)
		if err := wakeup(path.Join("/run", group.Name)); err != nil {
			h.logger.Errorw("failed to wake up", "stream", h.stream, "group", group.Name, "error", err)
		}
	}
	// ack by default if returned nil
	return nil
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

func getTopics(addr string) ([]nsqd.TopicStats, error) {
	// TODO: if nsqd would be embedded, we could use nsqd.GetStats() instead of http request
	resp, err := http.Get(fmt.Sprintf("http://%s/stats?format=json", addr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get stats")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}
	defer resp.Body.Close()

	result := struct {
		Topics []nsqd.TopicStats `json:"topics"`
	}{}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal stats")
	}

	return result.Topics, nil
}

func activeListeners(topics []nsqd.TopicStats) map[string][]api.Group {
	result := make(map[string][]api.Group)
	for _, topic := range topics {
		for _, channel := range topic.Channels {
			if utils.IsSocket(path.Join("/run", channel.ChannelName)) {
				result[topic.TopicName] = append(result[topic.TopicName], api.Group{
					Name:           channel.ChannelName,
					HasUnprocessed: channel.Depth > 0,
				})
			}
		}
	}
	return result
}
