package consts

const (
	TRACE_ID               = "Trace-Id"
	HOST_KEY               = "host"
	RID_KEY                = "rid"
	USER_ID_KEY            = "user_id"
	APP_ID_KEY             = "app_id"
	SECRET_KEY             = "sk"
	APP_IS_LIMIT_QUOTA_KEY = "app_is_limit_quota"
	KEY_IS_LIMIT_QUOTA_KEY = "key_is_limit_quota"
)

const (
	SESSION_RESELLER           = "session_reseller"
	SESSION_USER               = "session_user"
	SESSION_APP                = "session_app"
	SESSION_APP_KEY            = "session_app_key"
	SESSION_ERROR_MODEL_AGENTS = "session_error_model_agents"
	SESSION_ERROR_KEYS         = "session_error_keys"
)

const (
	DEFAULT_MODEL      = "gpt-3.5-turbo"
	QUOTA_DEFAULT_UNIT = 1000000.0 // $1 = 1M Tokens

	COMPLETION_ID_PREFIX = "chatcmpl-"
	COMPLETION_OBJECT    = "chat.completion"

	DELTA_TYPE_INPUT_JSON = "input_json_delta"
)
