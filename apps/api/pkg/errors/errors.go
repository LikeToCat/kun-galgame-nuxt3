package errors

import "fmt"

// AppError is the unified error type for all API responses.
// It implements the error interface.
type AppError struct {
	Code       int    `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func New(code int, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// Error codes compatible with the existing frontend (responseHandler.ts).
//
// 205 → authentication failure → client redirects to /login
// 233 → generic business error → client shows error message
const (
	CodeOK   = 0
	CodeAuth = 205
	CodeBiz  = 233
)

// Auth errors
func ErrUnauthorized(msg string) *AppError {
	return New(CodeAuth, msg, 401)
}

func ErrAuthExpired() *AppError {
	return New(CodeAuth, "用户登录失效", 401)
}

func ErrForbidden(msg string) *AppError {
	return New(CodeBiz, msg, 403)
}

// Business errors
func ErrBadRequest(msg string) *AppError {
	return New(CodeBiz, msg, 400)
}

func ErrNotFound(msg string) *AppError {
	return New(CodeBiz, msg, 404)
}

func ErrInternal(msg string) *AppError {
	return New(CodeBiz, msg, 500)
}

func ErrValidation(msg string) *AppError {
	return New(CodeBiz, msg, 400)
}
