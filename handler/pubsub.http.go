package handler

import (
	"golang-redis/errs"
	"golang-redis/service"

	"github.com/gofiber/fiber/v2"
)

type pubSubHandler struct {
	pubsubSvc service.PubSubService
}

func NewPubSubHandler(pubsubSvc service.PubSubService) pubSubHandler {
	return pubSubHandler{pubsubSvc: pubsubSvc}
}

func (h pubSubHandler) Publish(c *fiber.Ctx) error {
	var req service.PublishRequest
	if err := c.BodyParser(&req); err != nil {
		return handleError(c, errs.NewValidationError("invalid request body"))
	}

	resp, err := h.pubsubSvc.Publish(c.Context(), req)
	if err != nil {
		return handleError(c, err)
	}

	return sendSuccess(c, fiber.StatusOK, "message published", resp)
}
