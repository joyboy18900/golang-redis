package handler

import (
	"golang-redis/errs"
	"golang-redis/service"

	"github.com/gofiber/fiber/v2"
)

type lockHandler struct {
	lockSvc service.LockService
}

func NewLockHandler(lockSvc service.LockService) lockHandler {
	return lockHandler{lockSvc: lockSvc}
}

func (h lockHandler) RunCriticalSection(c *fiber.Ctx) error {
	var req service.RunCriticalSectionRequest
	if err := c.BodyParser(&req); err != nil {
		return handleError(c, errs.NewValidationError("invalid request body"))
	}

	resp, err := h.lockSvc.RunCriticalSection(c.Context(), req)
	if err != nil {
		return handleError(c, err)
	}

	return sendSuccess(c, fiber.StatusOK, "critical section completed", resp)
}
