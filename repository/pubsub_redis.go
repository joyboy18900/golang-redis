package repository

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type pubSubRepositoryRedis struct {
	client *redis.Client
}

func NewPubSubRepositoryRedis(client *redis.Client) PubSubRepository {
	return pubSubRepositoryRedis{client: client}
}

func (r pubSubRepositoryRedis) Publish(ctx context.Context, channel, payload string) (int64, error) {
	n, err := r.client.Publish(ctx, channel, payload).Result()
	if err != nil {
		return 0, fmt.Errorf("publish %s: %w", channel, err)
	}
	return n, nil
}

func (r pubSubRepositoryRedis) Subscribe(ctx context.Context, channel string) (<-chan Message, func() error, error) {
	ps := r.client.Subscribe(ctx, channel)
	if _, err := ps.Receive(ctx); err != nil {
		return nil, nil, fmt.Errorf("subscribe %s: %w", channel, err)
	}

	out := make(chan Message)
	redisCh := ps.Channel()

	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-redisCh:
				if !ok {
					return
				}
				select {
				case out <- Message{Channel: msg.Channel, Payload: msg.Payload}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, ps.Close, nil
}
