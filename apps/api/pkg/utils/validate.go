package utils

import (
	"kun-galgame-api/pkg/errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

var validate = validator.New()

// ParseAndValidate parses the request body into dst and validates it.
func ParseAndValidate(c *fiber.Ctx, dst any) *errors.AppError {
	if err := c.BodyParser(dst); err != nil {
		return errors.ErrBadRequest("请求格式错误")
	}
	if err := validate.Struct(dst); err != nil {
		return errors.ErrValidation(err.Error())
	}
	return nil
}

// ParseQueryAndValidate parses query params into dst and validates it.
func ParseQueryAndValidate(c *fiber.Ctx, dst any) *errors.AppError {
	if err := c.QueryParser(dst); err != nil {
		return errors.ErrBadRequest("查询参数格式错误")
	}
	if err := validate.Struct(dst); err != nil {
		return errors.ErrValidation(err.Error())
	}
	return nil
}
