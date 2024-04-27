package consts

const (
	SESSION_USER = "session_user"
	SESSION_APP  = "session_app"
	SESSION_KEY  = "session_key"

	USER_ID_KEY            = "user_id"
	APP_ID_KEY             = "app_id"
	SECRET_KEY             = "sk"
	APP_IS_LIMIT_QUOTA_KEY = "app_is_limit_quota"
	KEY_IS_LIMIT_QUOTA_KEY = "key_is_limit_quota"

	CORP_OPENAI     = "OpenAI"
	CORP_BAIDU      = "Baidu"
	CORP_XFYUN      = "Xfyun"
	CORP_ALIYUN     = "Aliyun"
	CORP_ZHIPUAI    = "ZhipuAI"
	CORP_MIDJOURNEY = "Midjourney"

	QUOTA_USD_UNIT = 500 * 1000.0 // $1 = 50万tokens

	ROLE_SYSTEM    = "system"
	ROLE_USER      = "user"
	ROLE_ASSISTANT = "assistant"
	ROLE_FUNCTION  = "function"
	ROLE_TOOL      = "tool"
)
