package errors

import (
	"errors"
	"fmt"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/sashabaranov/go-openai"
)

type IFastAPIError interface {
	Unwrap() error
	Status() int
	ErrCode() any
	ErrMessage() string
	ErrType() string
}

type FastAPIError struct {
	Err *openai.APIError `json:"error,omitempty"`
}

var (
	ERR_NIL                = NewError(500, -1, "", "fastapi_error")
	ERR_UNKNOWN            = NewError(500, -1, "Unknown Error", "fastapi_error")
	ERR_INTERNAL_ERROR     = NewError(500, 500, "Internal Error", "fastapi_error")
	ERR_NOT_AUTHORIZED     = NewError(403, "fastapi_error", "Not Authorized", "fastapi_error")
	ERR_NOT_FOUND          = NewError(404, "unknown_url", "Unknown request URL", "invalid_request_error")
	ERR_PERMISSION_DENIED  = NewError(401, "fastapi_request_error", "Unauthorized", "fastapi_request_error")
	ERR_NOT_API_KEY        = NewError(401, "invalid_request_error", "You didn't provide an API key.", "invalid_request_error")
	ERR_INVALID_API_KEY    = NewError(401, "invalid_api_key", "Incorrect API key provided.", "invalid_request_error")
	ERR_INSUFFICIENT_QUOTA = NewError(429, "insufficient_quota", "You exceeded your current quota.", "insufficient_quota")
)

func New(text string) error {
	return errors.New(text)
}

func Newf(format string, args ...interface{}) error {
	return gerror.Newf(format, args...)
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}

func As(err error, target any) bool {
	return errors.As(err, target)
}

func NewError(status int, code any, message, typ string) error {
	return &FastAPIError{
		Err: &openai.APIError{
			HTTPStatusCode: status,
			Code:           code,
			Message:        message,
			Type:           typ,
		},
	}
}

func Error(err error) IFastAPIError {

	if err == nil {
		return ERR_NIL.(IFastAPIError)
	}

	if e, ok := err.(IFastAPIError); ok {
		return e
	}

	e := &openai.APIError{}
	if errors.As(err, &e) {
		return NewError(e.HTTPStatusCode, e.Code, e.Message, e.Type).(IFastAPIError)
	}

	return NewError(200, "fastapi_error", err.Error(), "fastapi_error").(IFastAPIError)
}

func (e *FastAPIError) Error() string {

	if e.Err.HTTPStatusCode > 0 {
		return fmt.Sprintf("error, status code: %d, message: %s", e.Err.HTTPStatusCode, e.Err.Message)
	}

	return e.Err.Message
}

func (e *FastAPIError) Unwrap() error {
	return e.Err
}

func (e *FastAPIError) Status() int {
	return e.Err.HTTPStatusCode
}

func (e *FastAPIError) ErrCode() any {
	return e.Err.Code
}

func (e *FastAPIError) ErrMessage() string {
	return e.Err.Message
}

func (e *FastAPIError) ErrType() string {
	return e.Err.Type
}
