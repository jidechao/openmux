package errors

import "fmt"

// ErrorCode 错误码
type ErrorCode string

const (
	ErrCodeInvalidAPIKey     ErrorCode = "invalid_api_key"
	ErrCodeRateLimitExceeded ErrorCode = "rate_limit_exceeded"
	ErrCodeModelNotFound     ErrorCode = "model_not_found"
	ErrCodeProviderError     ErrorCode = "provider_error"
	ErrCodeNoAvailableBackend ErrorCode = "no_available_backend"
	ErrCodeInvalidRequest    ErrorCode = "invalid_request"
	ErrCodeTimeout           ErrorCode = "timeout"
)

// Error 自定义错误
type Error struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// New 创建新错误
func New(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装错误
func Wrap(code ErrorCode, message string, err error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsRetryable 判断错误是否可重试
func IsRetryable(err error) bool {
	if e, ok := err.(*Error); ok {
		switch e.Code {
		case ErrCodeProviderError, ErrCodeTimeout, ErrCodeNoAvailableBackend:
			return true
		}
	}
	return false
}

// IsRateLimitError 判断是否为限流错误
func IsRateLimitError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrCodeRateLimitExceeded
	}
	return false
}
