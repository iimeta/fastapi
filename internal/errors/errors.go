package errors

import (
	"context"
	"errors"
	"fmt"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/iimeta/fastapi-sdk/sdkerr"
)

type IFastApiError interface {
	Unwrap() error
	Status() int
	ErrCode() any
	ErrMessage() string
	ErrType() string
}

type FastApiError struct {
	Err *sdkerr.ApiError `json:"error,omitempty"`
}

var (
	ERR_NIL                          = NewError(500, -1, "", "fastapi_error")
	ERR_UNKNOWN                      = NewError(500, -1, "Unknown Error", "fastapi_error")
	ERR_SYSTEM                       = NewError(500, -1, "System Error.", "fastapi_error")
	ERR_INTERNAL_ERROR               = NewError(500, 500, "Internal Error", "fastapi_error")
	ERR_INVALID_PARAMETER            = NewError(400, "fastapi_error", "Invalid Parameter.", "fastapi_error")
	ERR_UNSUPPORTED_FILE_FORMAT      = NewError(400, "fastapi_error", "Unsupported file format.", "fastapi_error")
	ERR_NOT_FOUND                    = NewError(404, "unknown_url", "Unknown request URL", "invalid_request_error")
	ERR_NO_AVAILABLE_KEY             = NewError(500, "fastapi_error", "No available key", "fastapi_error")
	ERR_NO_AVAILABLE_MODEL_AGENT     = NewError(500, "fastapi_error", "No available model agent", "fastapi_error")
	ERR_NO_AVAILABLE_MODEL_AGENT_KEY = NewError(500, "fastapi_error", "No available model agent key", "fastapi_error")
	ERR_NOT_AUTHORIZED               = NewError(403, "fastapi_error", "Not Authorized", "fastapi_error")
	ERR_NOT_API_KEY                  = NewError(401, "invalid_request_error", "You didn't provide an API key.", "invalid_request_error")
	ERR_INVALID_API_KEY              = NewError(401, "invalid_api_key", "Incorrect API key provided or has been disabled.", "fastapi_request_error")
	ERR_API_KEY_DISABLED             = NewError(401, "api_key_disabled", "Key has been disabled.", "fastapi_request_error")
	ERR_INVALID_USER                 = NewError(401, "invalid_user", "User does not exist or has been disabled.", "fastapi_request_error")
	ERR_USER_DISABLED                = NewError(401, "user_disabled", "User has been disabled.", "fastapi_request_error")
	ERR_INVALID_APP                  = NewError(401, "invalid_app", "App does not exist or has been disabled.", "fastapi_request_error")
	ERR_APP_DISABLED                 = NewError(401, "app_disabled", "App has been disabled.", "fastapi_error")
	ERR_MODEL_NOT_FOUND              = NewError(404, "model_not_found", "The model does not exist or you do not have access to it.", "fastapi_request_error")
	ERR_PATH_NOT_FOUND               = NewError(404, "path_not_found", "The path does not exist or you do not have access to it.", "fastapi_request_error")
	ERR_MODEL_DISABLED               = NewError(401, "model_disabled", "Model has been disabled.", "fastapi_request_error")
	ERR_INSUFFICIENT_QUOTA           = NewError(429, "insufficient_quota", "You exceeded your current quota.", "insufficient_quota")
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
	return &FastApiError{
		Err: &sdkerr.ApiError{
			HttpStatusCode: status,
			Code:           code,
			Message:        message,
			Type:           typ,
		},
	}
}

func NewErrorf(status int, code any, message, typ string, args ...interface{}) error {
	return &FastApiError{
		Err: &sdkerr.ApiError{
			HttpStatusCode: status,
			Code:           code,
			Message:        fmt.Sprintf(message, args...),
			Type:           typ,
		},
	}
}

func Error(ctx context.Context, err error) IFastApiError {

	if err == nil {
		return ERR_NIL.(IFastApiError)
	}

	// 屏蔽不想对外暴露的错误
	if Is(err, ERR_NO_AVAILABLE_KEY) || Is(err, ERR_NO_AVAILABLE_MODEL_AGENT) || Is(err, ERR_NO_AVAILABLE_MODEL_AGENT_KEY) {
		err = ERR_SYSTEM
	}

	if e, ok := err.(IFastApiError); ok {
		if e.ErrCode() == 0 {
			return NewError(e.Status(), e.ErrCode(), e.ErrMessage(), e.ErrType()).(IFastApiError)
		}
		return NewErrorf(e.Status(), e.ErrCode(), e.ErrMessage()+" TraceId: %s", e.ErrType(), gctx.CtxId(ctx)).(IFastApiError)
	}

	apiError := &sdkerr.ApiError{}
	if As(err, &apiError) && apiError.HttpStatusCode != 500 {
		return NewError(apiError.HttpStatusCode, apiError.Code, apiError.Message, apiError.Type).(IFastApiError)
	}

	// 未知的错误, 用统一描述处理
	e := ERR_UNKNOWN.(IFastApiError)
	return NewErrorf(e.Status(), e.ErrCode(), e.ErrMessage()+" TraceId: %s", e.ErrType(), gctx.CtxId(ctx)).(IFastApiError)
}

func (e *FastApiError) Error() string {

	if e.Err.HttpStatusCode > 0 {
		return fmt.Sprintf("error, status code: %d, message: %s", e.Err.HttpStatusCode, e.Err.Message)
	}

	return e.Err.Message
}

func (e *FastApiError) Unwrap() error {
	return e.Err
}

func (e *FastApiError) Status() int {
	return e.Err.HttpStatusCode
}

func (e *FastApiError) ErrCode() any {
	return e.Err.Code
}

func (e *FastApiError) ErrMessage() string {
	return e.Err.Message
}

func (e *FastApiError) ErrType() string {
	return e.Err.Type
}
