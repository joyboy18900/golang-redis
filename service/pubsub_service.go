package service

import (
	"context"

	"golang-redis/errs"
	"golang-redis/logs"
	"golang-redis/repository"
)

type pubSubService struct {
	repo repository.PubSubRepository
}

func NewPubSubService(repo repository.PubSubRepository) PubSubService {
	return pubSubService{repo: repo}
}

func (s pubSubService) Publish(ctx context.Context, req PublishRequest) (*PublishResponse, error) {
	if req.Channel == "" {
		return nil, errs.NewValidationError("channel is required")
	}
	if req.Payload == "" {
		return nil, errs.NewValidationError("payload is required")
	}

	count, err := s.repo.Publish(ctx, req.Channel, req.Payload)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	return &PublishResponse{Channel: req.Channel, Payload: req.Payload, SubscriberCount: count}, nil
}

func (s pubSubService) RunConsumer(ctx context.Context, channel, consumerName string) error {
	messages, closer, err := s.repo.Subscribe(ctx, channel)
	if err != nil {
		return err
	}
	defer closer()

	for msg := range messages {
		logs.Info(consumerName + " received on " + msg.Channel + ": " + msg.Payload)
	}

	return ctx.Err()
}
