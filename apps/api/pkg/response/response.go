package response

import (
	"kun-galgame-api/pkg/errors"

	"github.com/gofiber/fiber/v2"
)

// OK sends a successful response with data.
//
//	{ "code": 0, "message": "成功", "data": ... }
func OK(c *fiber.Ctx, data any) error {
	return c.JSON(fiber.Map{
		"code":    errors.CodeOK,
		"message": "成功",
		"data":    data,
	})
}

// OKMessage sends a successful response with a custom message and no data.
func OKMessage(c *fiber.Ctx, msg string) error {
	return c.JSON(fiber.Map{
		"code":    errors.CodeOK,
		"message": msg,
	})
}

// Error sends an error response derived from an AppError.
func Error(c *fiber.Ctx, err *errors.AppError) error {
	return c.Status(err.StatusCode).JSON(fiber.Map{
		"code":    err.Code,
		"message": err.Message,
	})
}

// Paginated sends a paginated list response.
//
//	{ "code": 0, "message": "成功", "data": { "items": [...], "total": 42 } }
func Paginated(c *fiber.Ctx, items any, total int64) error {
	return c.JSON(fiber.Map{
		"code":    errors.CodeOK,
		"message": "成功",
		"data": fiber.Map{
			"items": items,
			"total": total,
		},
	})
}
