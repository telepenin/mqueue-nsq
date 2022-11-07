package utils

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/coreos/go-systemd/activation"
	"github.com/go-redis/redis/v9"
)

//NewRedisClient create a new instance of client redis
func NewRedisClient() (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:6379", os.Getenv("REDIS_HOST")),
		Password: "",
		DB:       0, // use default DB
	})

	ctx := context.TODO()
	_, err := client.Ping(ctx).Result()
	return client, err
}

// getSystemdSocket returns a socket passed through systemd socket activation.
// Returned listener may be nil if no sockets were passed.
func getSystemdSocket() ([]net.Listener, error) {
	listeners, err := activation.Listeners()
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve listeners: %s", err)
	}
	if len(listeners) == 0 {
		return nil, fmt.Errorf("no socket passed through systemd")
	} else if len(listeners) != 2 {
		return nil, fmt.Errorf("unexpected number of passed sockets: (%d != 2)", len(listeners))
	} else {
		return listeners, nil
	}
}
