package consts

const (
	API_USAGE_KEY = "api:%d:usage"

	USER_STATUS_FIELD       = "user.status"
	USER_USAGE_COUNT_FIELD  = "user.usage_count"
	USER_USED_TOKENS_FIELD  = "user.used_tokens"
	USER_TOTAL_TOKENS_FIELD = "user.total_tokens"

	APP_STATUS_FIELD         = "app.%d.status"
	APP_USAGE_COUNT_FIELD    = "app.%d.usage_count"
	APP_USED_TOKENS_FIELD    = "app.%d.used_tokens"
	APP_TOTAL_TOKENS_FIELD   = "app.%d.total_tokens"
	APP_IS_LIMIT_QUOTA_FIELD = "app.%d.is_limit_quota"

	KEY_STATUS_FIELD         = "key.%d.%s.status"
	KEY_USAGE_COUNT_FIELD    = "key.%d.%s.usage_count"
	KEY_USED_TOKENS_FIELD    = "key.%d.%s.used_tokens"
	KEY_TOTAL_TOKENS_FIELD   = "key.%d.%s.total_tokens"
	KEY_IS_LIMIT_QUOTA_FIELD = "key.%d.%s.is_limit_quota"

	MODEL_STATUS_FIELD = "model.%s.status"

	MODEL_AGENT_STATUS_FIELD     = "model:agent.%s.status"
	MODEL_AGENT_KEY_STATUS_FIELD = "model:agent.key.%s.status"
)

const (
	API_USER_KEY        = "api:user:%d"
	API_APP_KEY         = "api:app:%d"
	API_KEY_KEY         = "api:key:%s"
	API_MODEL_KEY       = "api:model:%s"
	API_MODEL_AGENT_KEY = "api:model:agent:%s"
)

const (
	API_USERS_KEY            = "api:users"
	API_APPS_KEY             = "api:apps"
	API_KEYS_KEY             = "api:keys"
	API_MODELS_KEY           = "api:models"
	API_MODEL_KEYS_KEY       = "api:model:keys:%s"
	API_MODEL_AGENTS_KEY     = "api:model_agents"
	API_MODEL_AGENT_KEYS_KEY = "api:model_agent:keys:%s"
)

const (
	SESSION_USER = "session_user"
	SESSION_APP  = "session_app"
	SESSION_KEY  = "session_key"
)

const (
	ERROR_MODEL_KEY       = "api:error:model:key:%s"
	ERROR_MODEL_AGENT     = "api:error:model:agent:%s"
	ERROR_MODEL_AGENT_KEY = "api:error:model:agent:key:%s"
)
