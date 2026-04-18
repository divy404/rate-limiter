package store

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)


type RedisStore struct {
	Client *redis.Client
}

func NewRedisStore(addr string) (*RedisStore,error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
		Password: "",
		DB: 0,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis at %s:%w", addr,err)
	}
	fmt.Printf("Connected to Redis at %s\n", addr)
	return &RedisStore{Client: client}, nil
}
func (r *RedisStore) Close() error {
	return r.Client.Close()
}