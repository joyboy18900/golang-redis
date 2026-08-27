package repository

import "context"

type Message struct {
	Channel string
	Payload string
}

//go:generate go tool mockgen -destination=../mock/mock_repository/pubsub.go golang-redis/repository PubSubRepository
type PubSubRepository interface {
	Publish(ctx context.Context, channel, payload string) (int64, error)
	Subscribe(ctx context.Context, channel string) (<-chan Message, func() error, error)
}
