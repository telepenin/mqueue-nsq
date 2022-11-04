package main

import (
	"go.uber.org/zap"

	"mqueue/pkg/utils"
)

func main() {
	zapLogger, _ := zap.NewDevelopment()
	defer zapLogger.Sync()
	logger := zapLogger.Sugar()

	logger.Info("wakeup is starting...")

	client, err := utils.NewRedisClient()
	if err != nil {
		logger.Error("failed to connect to redis", err)
		return
	}

	_ = client
}
