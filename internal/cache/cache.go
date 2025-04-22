package cache

import (
	"context"

	"github.com/redis/go-redis/v9"
)

func New(addr string, password string, DB int, Protocol int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // No password set
		DB:       0,  // Use default DB
		Protocol: 2,  // Connection protocol
	})

	ctx := context.Background()

	err := client.Set(ctx, "foo", "bar", 0).Err()
	if err != nil {
		return nil, err
	}

	_, err = client.Get(ctx, "foo").Result()
	if err != nil {
		return nil, err
	}

	client.Del(ctx, "foo")

	return client, nil
}
