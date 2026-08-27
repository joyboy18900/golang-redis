package service

import "context"

type PublishRequest struct {
	Channel string `json:"channel"`
	Payload string `json:"payload"`
}

type PublishResponse struct {
	Channel         string `json:"channel"`
	Payload         string `json:"payload"`
	SubscriberCount int64  `json:"subscriber_count"`
}

//go:generate go tool mockgen -destination=../mock/mock_service/pubsub.go golang-redis/service PubSubService
type PubSubService interface {
	Publish(ctx context.Context, req PublishRequest) (*PublishResponse, error)
	RunConsumer(ctx context.Context, channel, consumerName string) error
}
