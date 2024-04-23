package consts

const (
	API_USAGE_KEY = "api:%d:usage"

	USER_QUOTA_FIELD = "user.quota"
	APP_QUOTA_FIELD  = "app.%d.quota"
	KEY_QUOTA_FIELD  = "key.%d.%s.quota"

	API_USER_KEY    = "api:user:%d"
	API_APP_KEY     = "api:app:%d"
	API_APP_KEY_KEY = "api:app:key:%s"

	API_MODELS_KEY           = "api:models"
	API_MODEL_KEYS_KEY       = "api:model:keys:%s"
	API_MODEL_AGENTS_KEY     = "api:model_agents"
	API_MODEL_AGENT_KEYS_KEY = "api:model_agent:keys:%s"

	ERROR_MODEL_KEY       = "api:error:model:key:%s"
	ERROR_MODEL_AGENT     = "api:error:model:agent:%s"
	ERROR_MODEL_AGENT_KEY = "api:error:model:agent:key:%s"

	ACCESS_TOKEN_KEY = "api:baidu:access_token:%s"
)

const (
	LOCK_USER_KEY = "api:lock:user:%d"
	LOCK_APP_KEY  = "api:lock:app:%d"
	LOCK_SK_KEY   = "api:lock:sk:%s"
)
