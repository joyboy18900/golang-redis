package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"golang-redis/handler"
	"golang-redis/mock/mock_service"
	"golang-redis/service"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

func newPubSubTestApp(pubsubSvc service.PubSubService) *fiber.App {
	app := fiber.New()
	pubsubHdlr := handler.NewPubSubHandler(pubsubSvc)
	app.Post("/messages", pubsubHdlr.Publish)
	return app
}

func TestPubSubHandler_Publish(t *testing.T) {
	req := service.PublishRequest{Channel: "notifications", Payload: "hello"}

	ctrl := gomock.NewController(t)
	pubsubSvc := mock_service.NewMockPubSubService(ctrl)
	pubsubSvc.EXPECT().Publish(gomock.Any(), req).
		Return(&service.PublishResponse{Channel: "notifications", Payload: "hello", SubscriberCount: 2}, nil)

	app := newPubSubTestApp(pubsubSvc)

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		t.Fatalf("encode request body: %v", err)
	}

	httpReq := httptest.NewRequest(fiber.MethodPost, "/messages", &buf)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(httpReq)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
}

func TestPubSubHandler_Publish_InvalidBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	pubsubSvc := mock_service.NewMockPubSubService(ctrl)
	app := newPubSubTestApp(pubsubSvc)

	httpReq := httptest.NewRequest(fiber.MethodPost, "/messages", bytes.NewReader([]byte("not-json")))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(httpReq)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusUnprocessableEntity)
	}
}
