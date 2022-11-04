package utils

import (
	"context"
	"fmt"
	"os"

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
