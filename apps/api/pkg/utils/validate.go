package utils

import (
	"fmt"
	"strings"

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
		return errors.ErrValidation(translateValidationErrors(err))
	}
	return nil
}

// ParseQueryAndValidate parses query params into dst and validates it.
func ParseQueryAndValidate(c *fiber.Ctx, dst any) *errors.AppError {
	if err := c.QueryParser(dst); err != nil {
		return errors.ErrBadRequest("查询参数格式错误")
	}
	if err := validate.Struct(dst); err != nil {
		return errors.ErrValidation(translateValidationErrors(err))
	}
	return nil
}

// translateValidationErrors converts validator errors to user-friendly
// Chinese messages, avoiding exposure of internal struct/field names.
func translateValidationErrors(err error) string {
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return "请求参数验证失败"
	}

	messages := make([]string, 0, len(validationErrors))
	for _, fe := range validationErrors {
		messages = append(messages, translateFieldError(fe))
	}
	return strings.Join(messages, "; ")
}

func translateFieldError(fe validator.FieldError) string {
	field := fe.Field()
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s 不能为空", field)
	case "email":
		return "邮箱格式不正确"
	case "min":
		return fmt.Sprintf("%s 长度不能小于 %s", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s 长度不能大于 %s", field, fe.Param())
	case "oneof":
		return fmt.Sprintf("%s 的值必须是 %s 之一", field, fe.Param())
	default:
		return fmt.Sprintf("%s 验证失败", field)
	}
}
