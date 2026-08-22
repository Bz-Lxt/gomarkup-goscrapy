package redisx

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func Open(ctx context.Context, addr, password string, db int) (*redis.Client, error) {
	cli := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  4 * time.Second,
		ReadTimeout:  4 * time.Second,
		WriteTimeout: 4 * time.Second,
		PoolSize:     32,
		MinIdleConns: 4,
	})
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := cli.Ping(pingCtx).Err(); err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("ping redis %s: %w", addr, err)
	}
	return cli, nil
}

func NewFromOptions(opt *redis.Options) *redis.Client {
	return redis.NewClient(opt)
}
