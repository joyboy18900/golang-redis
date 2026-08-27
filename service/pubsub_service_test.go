package service_test

import (
	"context"
	"errors"
	"testing"

	"golang-redis/errs"
	"golang-redis/mock/mock_repository"
	"golang-redis/repository"
	"golang-redis/service"

	"go.uber.org/mock/gomock"
)

func TestPubSubService_Publish(t *testing.T) {
	tests := []struct {
		name    string
		req     service.PublishRequest
		setup   func(repo *mock_repository.MockPubSubRepository)
		wantErr error
	}{
		{
			name: "success",
			req:  service.PublishRequest{Channel: "notifications", Payload: "hello"},
			setup: func(repo *mock_repository.MockPubSubRepository) {
				repo.EXPECT().Publish(gomock.Any(), "notifications", "hello").Return(int64(2), nil)
			},
		},
		{
			name:    "missing channel",
			req:     service.PublishRequest{Payload: "hello"},
			setup:   func(repo *mock_repository.MockPubSubRepository) {},
			wantErr: errs.AppError{Code: 422},
		},
		{
			name:    "missing payload",
			req:     service.PublishRequest{Channel: "notifications"},
			setup:   func(repo *mock_repository.MockPubSubRepository) {},
			wantErr: errs.AppError{Code: 422},
		},
		{
			name: "repository error",
			req:  service.PublishRequest{Channel: "notifications", Payload: "hello"},
			setup: func(repo *mock_repository.MockPubSubRepository) {
				repo.EXPECT().Publish(gomock.Any(), "notifications", "hello").Return(int64(0), errors.New("redis down"))
			},
			wantErr: errs.AppError{Code: 500},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mock_repository.NewMockPubSubRepository(ctrl)
			tc.setup(repo)

			svc := service.NewPubSubService(repo)
			got, err := svc.Publish(context.Background(), tc.req)

			if tc.wantErr != nil {
				var appErr errs.AppError
				if !errors.As(err, &appErr) || appErr.Code != tc.wantErr.(errs.AppError).Code {
					t.Fatalf("Publish() error = %v, want AppError code %d", err, tc.wantErr.(errs.AppError).Code)
				}
				return
			}

			if err != nil {
				t.Fatalf("Publish() unexpected error = %v", err)
			}
			if got.SubscriberCount != 2 {
				t.Fatalf("Publish() subscriber count = %d, want 2", got.SubscriberCount)
			}
		})
	}
}

func TestPubSubService_RunConsumer_ReturnsNilOnChannelClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_repository.NewMockPubSubRepository(ctrl)

	messages := make(chan repository.Message, 1)
	messages <- repository.Message{Channel: "notifications", Payload: "hi"}
	close(messages)

	closed := false
	closer := func() error {
		closed = true
		return nil
	}

	repo.EXPECT().Subscribe(gomock.Any(), "notifications").Return(messages, closer, nil)

	svc := service.NewPubSubService(repo)
	if err := svc.RunConsumer(context.Background(), "notifications", "consumer-1"); err != nil {
		t.Fatalf("RunConsumer() error = %v, want nil", err)
	}
	if !closed {
		t.Fatal("RunConsumer() did not call the closer")
	}
}

func TestPubSubService_RunConsumer_ReturnsCtxErrOnCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_repository.NewMockPubSubRepository(ctrl)

	messages := make(chan repository.Message)
	closerCalled := false
	closer := func() error {
		closerCalled = true
		return nil
	}

	repo.EXPECT().Subscribe(gomock.Any(), "notifications").Return((<-chan repository.Message)(messages), closer, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	close(messages)

	svc := service.NewPubSubService(repo)
	err := svc.RunConsumer(ctx, "notifications", "consumer-1")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunConsumer() error = %v, want context.Canceled", err)
	}
	if !closerCalled {
		t.Fatal("RunConsumer() did not call the closer")
	}
}
