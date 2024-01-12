package consts

const (
	USER_ID_KEY = "uid"
	APP_ID_KEY  = "app_id"
	SECRET_KEY  = "sk"
)

const (
	LOCK_SK_KEY = "api:lock:sk:%s"
)

const (
	API_USAGE_KEY      = "api:%d:usage"
	USAGE_COUNT_FIELD  = "%v.usage_count"
	USED_TOKENS_FIELD  = "%v.used_tokens"
	TOTAL_TOKENS_FIELD = "%v.total_tokens"

	USER_USAGE_COUNT_FIELD  = "user.%d.usage_count"
	USER_USED_TOKENS_FIELD  = "user.%d.used_tokens"
	USER_TOTAL_TOKENS_FIELD = "user.%d.total_tokens"

	APP_USAGE_COUNT_FIELD  = "app.%d.usage_count"
	APP_USED_TOKENS_FIELD  = "app.%d.used_tokens"
	APP_TOTAL_TOKENS_FIELD = "app.%d.total_tokens"

	KEY_USAGE_COUNT_FIELD  = "key.%s.usage_count"
	KEY_USED_TOKENS_FIELD  = "key.%s.used_tokens"
	KEY_TOTAL_TOKENS_FIELD = "key.%s.total_tokens"
)
