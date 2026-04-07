package middleware

import (
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v2"
)

// RequireRole creates a middleware that checks user role >= minRole.
func RequireRole(minRole int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user := GetUser(c)
		if user == nil {
			return response.Error(c, errors.ErrAuthExpired())
		}
		if user.Role < minRole {
			return response.Error(c, errors.ErrForbidden("您没有权限进行此操作"))
		}
		return c.Next()
	}
}
