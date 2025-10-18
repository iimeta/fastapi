package errors

import (
	"context"
	"errors"
	"fmt"

	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	serrors "github.com/iimeta/fastapi-sdk/errors"
	"github.com/iimeta/fastapi/internal/config"
)

type IFastApiError interface {
	Unwrap() error
	Status() int
	ErrCode() any
	ErrMessage() string
	ErrType() string
	ErrParam() any
}

type FastApiError struct {
	Err *serrors.ApiError `json:"error,omitempty"`
}

var (
	ERR_NIL                           = NewError(500, -1, "", "fastapi_error", nil)
	ERR_UNKNOWN                       = NewError(500, -1, "Unknown Error.", "fastapi_error", nil)
	ERR_INTERNAL_ERROR                = NewError(500, 500, "Internal Error.", "fastapi_error", nil)
	ERR_SERVICE_UNAVAILABLE           = NewError(503, 503, "Service Unavailable.", "fastapi_error", nil)
	ERR_NO_AVAILABLE_KEY              = NewError(500, "fastapi_error", "No available key.", "fastapi_error", nil)
	ERR_ALL_KEY                       = NewError(500, "fastapi_error", "All key error.", "fastapi_error", nil)
	ERR_NO_AVAILABLE_MODEL_AGENT      = NewError(500, "fastapi_error", "No available model agent.", "fastapi_error", nil)
	ERR_ALL_MODEL_AGENT               = NewError(500, "fastapi_error", "All model agent error.", "fastapi_error", nil)
	ERR_MODEL_AGENT_HAS_BEEN_DISABLED = NewError(500, "fastapi_error", "Model agent has been disabled.", "fastapi_error", nil)
	ERR_NO_AVAILABLE_MODEL_AGENT_KEY  = NewError(500, "fastapi_error", "No available model agent key.", "fastapi_error", nil)
	ERR_ALL_MODEL_AGENT_KEY           = NewError(500, "fastapi_error", "All model agent key error.", "fastapi_error", nil)
	ERR_MODEL_HAS_BEEN_DISABLED       = NewError(500, "fastapi_error", "Model has been disabled.", "fastapi_error", nil)
	ERR_INVALID_PARAMETER             = NewError(400, "invalid_parameter", "Invalid Parameter.", "fastapi_request_error", nil)
	ERR_UNSUPPORTED_FILE_FORMAT       = NewError(400, "unsupported_file_format", "Unsupported file format.", "fastapi_request_error", nil)
	ERR_NOT_API_KEY                   = NewError(401, "invalid_request_error", "You didn't provide an API key.", "fastapi_request_error", nil)
	ERR_INVALID_API_KEY               = NewError(401, "invalid_api_key", "Incorrect API key provided or has been disabled.", "fastapi_request_error", nil)
	ERR_API_KEY_DISABLED              = NewError(401, "api_key_disabled", "Key has been disabled.", "fastapi_request_error", nil)
	ERR_INVALID_RESELLER              = NewError(401, "invalid_reseller", "Reseller does not exist or has been disabled.", "fastapi_request_error", nil)
	ERR_RESELLER_DISABLED             = NewError(401, "reseller_disabled", "Reseller has been disabled.", "fastapi_request_error", nil)
	ERR_INVALID_USER                  = NewError(401, "invalid_user", "User does not exist or has been disabled.", "fastapi_request_error", nil)
	ERR_USER_DISABLED                 = NewError(401, "user_disabled", "User has been disabled.", "fastapi_request_error", nil)
	ERR_INVALID_APP                   = NewError(401, "invalid_app", "App does not exist or has been disabled.", "fastapi_request_error", nil)
	ERR_APP_DISABLED                  = NewError(401, "app_disabled", "App has been disabled.", "fastapi_request_error", nil)
	ERR_MODEL_DISABLED                = NewError(401, "model_disabled", "Model has been disabled.", "fastapi_request_error", nil)
	ERR_GROUP_DISABLED                = NewError(401, "group_disabled", "Group has been disabled.", "fastapi_request_error", nil)
	ERR_GROUP_EXPIRED                 = NewError(401, "group_expired", "Group has expired.", "fastapi_request_error", nil)
	ERR_FORBIDDEN                     = NewError(403, "forbidden", "Forbidden.", "fastapi_request_error", nil)
	ERR_NOT_AUTHORIZED                = NewError(403, "not_authorized", "Not Authorized.", "fastapi_request_error", nil)
	ERR_NOT_FOUND                     = NewError(404, "unknown_url", "Unknown request URL.", "fastapi_request_error", nil)
	ERR_MODEL_NOT_FOUND               = NewError(404, "model_not_found", "The model does not exist or you do not have access to it.", "fastapi_request_error", nil)
	ERR_PATH_NOT_FOUND                = NewError(404, "path_not_found", "The path does not exist or you do not have access to it.", "fastapi_request_error", nil)
	ERR_GROUP_NOT_FOUND               = NewError(404, "group_not_found", "The group does not exist or you do not have access to it.", "fastapi_request_error", nil)
	ERR_RESELLER_INSUFFICIENT_QUOTA   = NewError(429, "reseller_insufficient_quota", "You reseller exceeded current quota.", "fastapi_request_error", nil)
	ERR_RESELLER_QUOTA_EXPIRED        = NewError(429, "reseller_quota_expired", "You reseller quota has expired.", "fastapi_request_error", nil)
	ERR_INSUFFICIENT_QUOTA            = NewError(429, "insufficient_quota", "You exceeded your current quota.", "fastapi_request_error", nil)
	ERR_ACCOUNT_QUOTA_EXPIRED         = NewError(429, "account_quota_expired", "You account quota has expired.", "fastapi_request_error", nil)
	ERR_APP_QUOTA_EXPIRED             = NewError(429, "app_quota_expired", "You app quota has expired.", "fastapi_request_error", nil)
	ERR_KEY_QUOTA_EXPIRED             = NewError(429, "key_quota_expired", "You key quota has expired.", "fastapi_request_error", nil)
	ERR_GROUP_INSUFFICIENT_QUOTA      = NewError(429, "group_insufficient_quota", "Group exceeded current quota.", "fastapi_request_error", nil)
)

func NewError(status int, code any, message, typ string, param any) error {
	return &FastApiError{
		Err: &serrors.ApiError{
			HttpStatusCode: status,
			Code:           code,
			Message:        message,
			Type:           typ,
			Param:          param,
		},
	}
}

func NewErrorf(status int, code any, message, typ string, param any, args ...interface{}) error {
	return &FastApiError{
		Err: &serrors.ApiError{
			HttpStatusCode: status,
			Code:           code,
			Message:        fmt.Sprintf(message, args...),
			Type:           typ,
			Param:          param,
		},
	}
}

func Error(ctx context.Context, err error) (iFastApiError IFastApiError) {

	defer func() {
		if config.Cfg.Core.ErrorPrefix != "fastapi" {
			code := iFastApiError.ErrCode()
			if c, ok := code.(string); ok {
				code = gstr.Replace(c, "fastapi", config.Cfg.Core.ErrorPrefix)
			}
			iFastApiError = NewError(iFastApiError.Status(), code, iFastApiError.ErrMessage(), gstr.Replace(iFastApiError.ErrType(), "fastapi", config.Cfg.Core.ErrorPrefix), nil).(IFastApiError)
		}
	}()

	if err == nil {
		return ERR_NIL.(IFastApiError)
	}

	// 屏蔽不想对外暴露的错误
	if Is(err, ERR_NO_AVAILABLE_KEY) || Is(err, ERR_NO_AVAILABLE_MODEL_AGENT) ||
		Is(err, ERR_MODEL_AGENT_HAS_BEEN_DISABLED) || Is(err, ERR_NO_AVAILABLE_MODEL_AGENT_KEY) ||
		Is(err, ERR_ALL_KEY) || Is(err, ERR_ALL_MODEL_AGENT) ||
		Is(err, ERR_ALL_MODEL_AGENT_KEY) || Is(err, ERR_MODEL_HAS_BEEN_DISABLED) {
		err = ERR_INTERNAL_ERROR
	}

	if e, ok := err.(IFastApiError); ok {
		return NewErrorf(e.Status(), e.ErrCode(), e.ErrMessage()+" TraceId: %s Timestamp: %d", e.ErrType(), e.ErrParam(), gctx.CtxId(ctx), gtime.TimestampMilli()).(IFastApiError)
	}

	// 不屏蔽错误
	if config.Cfg.NotShieldError.Open && len(config.Cfg.NotShieldError.Errors) > 0 {
		for _, notShieldError := range config.Cfg.NotShieldError.Errors {
			if gstr.Contains(err.Error(), notShieldError) {

				e := ERR_UNKNOWN.(IFastApiError)

				requestError := &serrors.RequestError{}
				if As(err, &requestError) {
					return NewErrorf(requestError.HttpStatusCode, e.ErrCode(), gstr.Split(gstr.Split(requestError.Err.Error(), " TraceId")[0], " (request id:")[0]+" TraceId: %s Timestamp: %d", e.ErrType(), e.ErrParam(), gctx.CtxId(ctx), gtime.TimestampMilli()).(IFastApiError)
				}

				apiError := &serrors.ApiError{}
				if As(err, &apiError) {
					return NewErrorf(apiError.HttpStatusCode, apiError.Code, gstr.Split(gstr.Split(apiError.Message, " TraceId")[0], " (request id:")[0]+" TraceId: %s Timestamp: %d", apiError.Type, apiError.Param, gctx.CtxId(ctx), gtime.TimestampMilli()).(IFastApiError)
				}

				return NewErrorf(e.Status(), e.ErrCode(), gstr.Split(gstr.Split(err.Error(), " TraceId")[0], " (request id:")[0]+" TraceId: %s Timestamp: %d", e.ErrType(), e.ErrParam(), gctx.CtxId(ctx), gtime.TimestampMilli()).(IFastApiError)
			}
		}
	}

	// 未知的错误, 用统一描述处理
	e := ERR_UNKNOWN.(IFastApiError)

	return NewErrorf(e.Status(), e.ErrCode(), e.ErrMessage()+" TraceId: %s Timestamp: %d", e.ErrType(), e.ErrParam(), gctx.CtxId(ctx), gtime.TimestampMilli()).(IFastApiError)
}

func (e *FastApiError) Error() string {
	return fmt.Sprintf("statusCode: %d, code: %s, message: %s", e.Err.HttpStatusCode, e.Err.Code, e.Err.Message)
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

func (e *FastApiError) ErrParam() any {
	return e.Err.Param
}

func New(text string) error {
	return errors.New(text)
}

func Newf(format string, args ...interface{}) error {
	return errors.New(fmt.Sprintf(format, args...))
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}

func As(err error, target any) bool {
	return errors.As(err, target)
}
