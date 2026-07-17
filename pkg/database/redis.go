package database

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"
	"wealth-warden/pkg/config"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func ConnectToRedis(cfg *config.Config, zapLogger *zap.Logger) (*redis.Client, error) {
	addr := net.JoinHostPort(cfg.Redis.Host, strconv.Itoa(cfg.Redis.Port))

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		zapLogger.Error("Could not ping redis", zap.String("addr", addr), zap.Error(err))
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	zapLogger.Info("Connected to redis", zap.String("addr", addr))
	return client, nil
}
