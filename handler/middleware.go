package handler

import (
	"strconv"

	"golang-redis/errs"
	"golang-redis/service"

	"github.com/gofiber/fiber/v2"
)

func RateLimitMiddleware(rateLimiterSvc service.RateLimiterService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		decision, err := rateLimiterSvc.Allow(c.Context(), c.IP())
		if err != nil {
			return handleError(c, err)
		}

		c.Set("X-RateLimit-Remaining", strconv.Itoa(decision.Remaining))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(decision.ResetAt.Unix(), 10))

		if !decision.Allowed {
			return handleError(c, errs.NewTooManyRequestsError("rate limit exceeded"))
		}

		return c.Next()
	}
}
