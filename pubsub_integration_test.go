package main_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"golang-redis/repository"
	"golang-redis/service"
)

func TestPubSub_FanOutToMultipleConsumers(t *testing.T) {
	client := connectTestRedis(t)
	defer client.Close()

	channel := "notifications-" + t.Name() + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)

	repo := repository.NewPubSubRepositoryRedis(client)
	svc := service.NewPubSubService(repo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumer1 := make(chan repository.Message, 1)
	consumer2 := make(chan repository.Message, 1)

	messages1, close1, err := repo.Subscribe(ctx, channel)
	if err != nil {
		t.Fatalf("Subscribe() consumer 1 error = %v", err)
	}
	defer close1()

	messages2, close2, err := repo.Subscribe(ctx, channel)
	if err != nil {
		t.Fatalf("Subscribe() consumer 2 error = %v", err)
	}
	defer close2()

	go func() {
		if msg, ok := <-messages1; ok {
			consumer1 <- msg
		}
	}()
	go func() {
		if msg, ok := <-messages2; ok {
			consumer2 <- msg
		}
	}()

	resp, err := svc.Publish(context.Background(), service.PublishRequest{Channel: channel, Payload: "fan-out"})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if resp.SubscriberCount != 2 {
		t.Fatalf("SubscriberCount = %d, want 2", resp.SubscriberCount)
	}

	for i, ch := range []chan repository.Message{consumer1, consumer2} {
		select {
		case msg := <-ch:
			if msg.Payload != "fan-out" {
				t.Fatalf("consumer %d payload = %q, want %q", i+1, msg.Payload, "fan-out")
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("consumer %d timed out waiting for the message", i+1)
		}
	}
}
