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
// 234 → account banned → client shows banned page; DOES NOT redirect to
//       /login because logging in again hits the same OAuth 10014 error
//       (see docs/oauth/api-reference.md §错误码速查)
const (
	CodeOK      = 0
	CodeAuth    = 205
	CodeBiz     = 233
	CodeBanned  = 234
)

// Auth errors
func ErrUnauthorized(msg string) *AppError {
	return New(CodeAuth, msg, 401)
}

func ErrAuthExpired() *AppError {
	return New(CodeAuth, "用户登录失效", 401)
}

// ErrAccountBanned signals OAuth-side 10014. Distinct from ErrAuthExpired
// so the frontend can show a banned page rather than bouncing the user
// through /login → OAuth → /login in a loop.
func ErrAccountBanned() *AppError {
	return New(CodeBanned, "账号已封禁", 403)
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
