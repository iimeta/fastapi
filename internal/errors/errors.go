package errors

import (
	"context"
	"errors"
	"fmt"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/net/gtrace"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	serrors "github.com/iimeta/fastapi-sdk/v2/errors"
	"github.com/iimeta/fastapi/v2/internal/config"
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
	ERR_NIL                               = NewError(500, -1, "", "fastapi_error", nil)
	ERR_UNKNOWN                           = NewError(500, -1, "Unknown Error.", "fastapi_error", nil)
	ERR_INTERNAL_ERROR                    = NewError(500, 500, "Internal Error.", "fastapi_error", nil)
	ERR_SERVICE_UNAVAILABLE               = NewError(503, 503, "Service Unavailable.", "fastapi_error", nil)
	ERR_NO_AVAILABLE_KEY                  = NewError(500, "fastapi_error", "No available key.", "fastapi_error", nil)
	ERR_ALL_KEY                           = NewError(500, "fastapi_error", "All key error.", "fastapi_error", nil)
	ERR_NO_AVAILABLE_MODEL_AGENT          = NewError(500, "fastapi_error", "No available model agent.", "fastapi_error", nil)
	ERR_GROUP_NO_AVAILABLE_MODEL_AGENT    = NewError(500, "fastapi_error", "The model agent for this model is not in this group.", "fastapi_error", nil)
	ERR_ALL_MODEL_AGENT                   = NewError(500, "fastapi_error", "All model agent error.", "fastapi_error", nil)
	ERR_MODEL_AGENT_HAS_BEEN_DISABLED     = NewError(500, "fastapi_error", "Model agent has been disabled.", "fastapi_error", nil)
	ERR_NO_AVAILABLE_MODEL_AGENT_KEY      = NewError(500, "fastapi_error", "No available model agent key.", "fastapi_error", nil)
	ERR_ALL_MODEL_AGENT_KEY               = NewError(500, "fastapi_error", "All model agent key error.", "fastapi_error", nil)
	ERR_MODEL_HAS_BEEN_DISABLED           = NewError(500, "fastapi_error", "Model has been disabled.", "fastapi_error", nil)
	ERR_GROUP_IS_NIL                      = NewError(500, "fastapi_error", "Group is nil.", "fastapi_error", nil)
	ERR_MISSING_REQUIRED_PARAMETER_IMAGE  = NewError(400, "missing_required_parameter", "Missing required parameter: 'image'.", "invalid_request_error", "image")
	ERR_MISSING_REQUIRED_PARAMETER_IMAGES = NewError(400, "missing_required_parameter", "Missing required parameter: 'images'.", "invalid_request_error", "images")
	ERR_UNSUPPORTED_FILE_FORMAT           = NewError(400, "unsupported_file_format", "Unsupported file format.", "fastapi_request_error", nil)
	ERR_UNSUPPORTED_BILLING_METHOD_MODEL  = NewError(400, "unsupported_billing_method_model", "Billing methods not supported by the current model.", "fastapi_request_error", nil)
	ERR_UNSUPPORTED_BILLING_METHOD_GROUP  = NewError(400, "unsupported_billing_method_group", "Billing methods not supported by the current group.", "fastapi_request_error", nil)
	ERR_UNSUPPORTED_ENDPOINT              = NewError(400, "unsupported_endpoint", "This endpoint is not supported by the current model.", "fastapi_request_error", nil)
	ERR_NOT_API_KEY                       = NewError(401, "invalid_request_error", "You didn't provide an API key.", "fastapi_request_error", nil)
	ERR_INVALID_API_KEY                   = NewError(401, "invalid_api_key", "Incorrect API key provided or has been disabled.", "fastapi_request_error", nil)
	ERR_API_KEY_DISABLED                  = NewError(401, "api_key_disabled", "Key has been disabled.", "fastapi_request_error", nil)
	ERR_INVALID_RESELLER                  = NewError(401, "invalid_reseller", "Reseller does not exist or has been disabled.", "fastapi_request_error", nil)
	ERR_RESELLER_DISABLED                 = NewError(401, "reseller_disabled", "Reseller has been disabled.", "fastapi_request_error", nil)
	ERR_INVALID_USER                      = NewError(401, "invalid_user", "User does not exist or has been disabled.", "fastapi_request_error", nil)
	ERR_USER_DISABLED                     = NewError(401, "user_disabled", "User has been disabled.", "fastapi_request_error", nil)
	ERR_USER_EXPIRED                      = NewError(401, "user_expired", "User has expired.", "fastapi_request_error", nil)
	ERR_INVALID_APP                       = NewError(401, "invalid_app", "App does not exist or has been disabled.", "fastapi_request_error", nil)
	ERR_APP_DISABLED                      = NewError(401, "app_disabled", "App has been disabled.", "fastapi_request_error", nil)
	ERR_MODEL_DISABLED                    = NewError(401, "model_disabled", "Model has been disabled.", "fastapi_request_error", nil)
	ERR_GROUP_DISABLED                    = NewError(401, "group_disabled", "Group has been disabled.", "fastapi_request_error", nil)
	ERR_GROUP_EXPIRED                     = NewError(401, "group_expired", "Group has expired.", "fastapi_request_error", nil)
	ERR_FORBIDDEN                         = NewError(403, "forbidden", "Forbidden.", "fastapi_request_error", nil)
	ERR_NOT_AUTHORIZED                    = NewError(403, "not_authorized", "Not Authorized.", "fastapi_request_error", nil)
	ERR_NOT_FOUND                         = NewError(404, "unknown_url", "Unknown request URL.", "fastapi_request_error", nil)
	ERR_MODEL_NOT_FOUND                   = NewError(404, "model_not_found", "The model does not exist or you do not have access to it.", "fastapi_request_error", nil)
	ERR_PATH_NOT_FOUND                    = NewError(404, "path_not_found", "The path does not exist or you do not have access to it.", "fastapi_request_error", nil)
	ERR_GROUP_NOT_FOUND                   = NewError(404, "group_not_found", "The group does not exist or you do not have access to it.", "fastapi_request_error", nil)
	ERR_RESELLER_INSUFFICIENT_QUOTA       = NewError(429, "reseller_insufficient_quota", "You reseller exceeded current quota.", "fastapi_request_error", nil)
	ERR_RESELLER_QUOTA_EXPIRED            = NewError(429, "reseller_quota_expired", "You reseller quota has expired.", "fastapi_request_error", nil)
	ERR_INSUFFICIENT_QUOTA                = NewError(429, "insufficient_quota", "You exceeded your current quota.", "fastapi_request_error", nil)
	ERR_ACCOUNT_QUOTA_EXPIRED             = NewError(429, "account_quota_expired", "You account quota has expired.", "fastapi_request_error", nil)
	ERR_APP_QUOTA_EXPIRED                 = NewError(429, "app_quota_expired", "You app quota has expired.", "fastapi_request_error", nil)
	ERR_KEY_QUOTA_EXPIRED                 = NewError(429, "key_quota_expired", "You key quota has expired.", "fastapi_request_error", nil)
	ERR_GROUP_INSUFFICIENT_QUOTA          = NewError(429, "group_insufficient_quota", "Group exceeded current quota.", "fastapi_request_error", nil)
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

func NewErrorf(status int, code any, message, typ string, param any, args ...any) error {
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

		if len(config.Cfg.Core.ReplaceErrorPrefixes) == 0 && config.Cfg.Core.ErrorPrefix == "fastapi" {
			return
		}

		for _, prefix := range append(config.Cfg.Core.ReplaceErrorPrefixes, "fastapi") {
			code := iFastApiError.ErrCode()
			if c, ok := code.(string); ok {
				code = gstr.Replace(c, prefix, config.Cfg.Core.ErrorPrefix)
			}
			iFastApiError = NewError(iFastApiError.Status(), code, iFastApiError.ErrMessage(), gstr.Replace(iFastApiError.ErrType(), prefix, config.Cfg.Core.ErrorPrefix), nil).(IFastApiError)
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
		return NewErrorf(e.Status(), e.ErrCode(), e.ErrMessage()+" TraceId: %s Timestamp: %d", e.ErrType(), e.ErrParam(), gtrace.GetTraceID(ctx), gtime.TimestampMilli()).(IFastApiError)
	}

	// 不屏蔽错误
	if config.Cfg.NotShieldError.Open && len(config.Cfg.NotShieldError.Errors) > 0 {
		for _, notShieldError := range config.Cfg.NotShieldError.Errors {
			if gstr.Contains(err.Error(), notShieldError) {

				e := ERR_UNKNOWN.(IFastApiError)

				requestError := &serrors.RequestError{}
				if As(err, &requestError) {
					return NewErrorf(requestError.HttpStatusCode, e.ErrCode(), gstr.Split(gstr.Split(requestError.Err.Error(), " TraceId")[0], " (request id:")[0]+" TraceId: %s Timestamp: %d", e.ErrType(), e.ErrParam(), gtrace.GetTraceID(ctx), gtime.TimestampMilli()).(IFastApiError)
				}

				apiError := &serrors.ApiError{}
				if As(err, &apiError) {
					return NewErrorf(apiError.HttpStatusCode, apiError.Code, gstr.Split(gstr.Split(apiError.Message, " TraceId")[0], " (request id:")[0]+" TraceId: %s Timestamp: %d", apiError.Type, apiError.Param, gtrace.GetTraceID(ctx), gtime.TimestampMilli()).(IFastApiError)
				}

				return NewErrorf(e.Status(), e.ErrCode(), gstr.Split(gstr.Split(err.Error(), " TraceId")[0], " (request id:")[0]+" TraceId: %s Timestamp: %d", e.ErrType(), e.ErrParam(), gtrace.GetTraceID(ctx), gtime.TimestampMilli()).(IFastApiError)
			}
		}
	}

	// 未知的错误, 用统一描述处理
	e := ERR_UNKNOWN.(IFastApiError)

	return NewErrorf(e.Status(), e.ErrCode(), e.ErrMessage()+" TraceId: %s Timestamp: %d", e.ErrType(), e.ErrParam(), gtrace.GetTraceID(ctx), gtime.TimestampMilli()).(IFastApiError)
}

// 按"不屏蔽错误"配置处理需对外暴露的任务错误信息(如异步任务失败原因):
// 命中不屏蔽名单时保留原始 code, message 清理已有 TraceId/request id 后追加新的 TraceId/Timestamp 返回;
// 否则视为屏蔽错误, 统一返回未知错误(ERR_UNKNOWN)
func ShieldError(ctx context.Context, code, message string) (string, string) {

	// 不屏蔽错误
	if config.Cfg.NotShieldError.Open && len(config.Cfg.NotShieldError.Errors) > 0 {
		for _, notShieldError := range config.Cfg.NotShieldError.Errors {
			if gstr.Contains(message, notShieldError) {
				return code, fmt.Sprintf("%s TraceId: %s Timestamp: %d", gstr.Split(gstr.Split(message, " TraceId")[0], " (request id:")[0], gtrace.GetTraceID(ctx), gtime.TimestampMilli())
			}
		}
	}

	// 屏蔽的错误, 用统一描述处理
	e := ERR_UNKNOWN.(IFastApiError)

	return fmt.Sprintf("%v", e.ErrCode()), fmt.Sprintf("%s TraceId: %s Timestamp: %d", e.ErrMessage(), gtrace.GetTraceID(ctx), gtime.TimestampMilli())
}

// 处理任务查询类响应JSON中各常见位置的错误信息(如厂商视频任务查询的官方透传响应):
// 支持顶层及 error、task.error、output 下的 code/message 字段(bailian/volcengine/xai/minimax),
// 以及 minimax 风格的 base_resp.status_code/status_msg 信封;
// 命中"不屏蔽错误"名单的原样返回(保留原始 code), 否则统一替换为未知错误(ERR_UNKNOWN)
func ShieldErrorJson(ctx context.Context, responseBytes []byte) []byte {

	if len(responseBytes) == 0 {
		return responseBytes
	}

	j, err := gjson.DecodeToJson(responseBytes)
	if err != nil {
		return responseBytes
	}

	shielded := false

	// code/message 成对出现的错误字段路径(空串为顶层)
	for _, path := range []string{"", "error", "task.error", "output"} {

		prefix := path
		if prefix != "" {
			prefix += "."
		}

		message := j.Get(prefix + "message").String()
		if message == "" {
			continue
		}

		code, msg := ShieldError(ctx, j.Get(prefix+"code").String(), message)
		if err = j.Set(prefix+"code", code); err != nil {
			return responseBytes
		}
		if err = j.Set(prefix+"message", msg); err != nil {
			return responseBytes
		}

		shielded = true
	}

	// minimax 风格的 base_resp 信封: status_code 非 0 且 status_msg 非空时按错误处理
	if statusMsg := j.Get("base_resp.status_msg").String(); statusMsg != "" {

		if statusCode := j.Get("base_resp.status_code").Int(); statusCode != 0 {

			code, msg := ShieldError(ctx, gconv.String(statusCode), statusMsg)

			// status_code 保持数值类型: 不屏蔽时 ShieldError 返回原值, 屏蔽时返回 "-1"
			if newCode := gconv.Int(code); newCode != statusCode {
				if err = j.Set("base_resp.status_code", newCode); err != nil {
					return responseBytes
				}
			}
			if err = j.Set("base_resp.status_msg", msg); err != nil {
				return responseBytes
			}

			shielded = true
		}
	}

	if !shielded {
		return responseBytes
	}

	data, err := j.ToJson()
	if err != nil {
		return responseBytes
	}

	return data
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

func Newf(format string, args ...any) error {
	return errors.New(fmt.Sprintf(format, args...))
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}

func As(err error, target any) bool {
	return errors.As(err, target)
}
