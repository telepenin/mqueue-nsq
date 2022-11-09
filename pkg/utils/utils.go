package utils

import (
	"context"
	"fmt"
	"io/fs"
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

// GetSystemdSocket returns a socket passed through systemd socket activation.
// Returned listener may be nil if no sockets were passed.
func GetSystemdSocket() (net.Listener, error) {
	listeners, err := activation.Listeners()
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve listeners: %s", err)
	}
	if len(listeners) == 0 {
		// probably we are not under systemd
		return nil, nil
	} else if len(listeners) != 1 {
		return nil, fmt.Errorf("unexpected number of passed sockets: (%d != 1)", len(listeners))
	} else {
		return listeners[0], nil
	}
}

// IsSocket returns true if the given path is a socket.
func IsSocket(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.Mode().Type() == fs.ModeSocket
}
