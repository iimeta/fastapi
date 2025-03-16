package consts

const (
	API_USAGE_KEY = "api:user:%d:usage"

	USER_QUOTA_FIELD = "user.quota"
	APP_QUOTA_FIELD  = "app.%d.quota"
	KEY_QUOTA_FIELD  = "key.%d.%s.quota"

	ERROR_MODEL_KEY       = "api:error:model:key:%s"
	ERROR_MODEL_AGENT     = "api:error:model:agent:%s"
	ERROR_MODEL_AGENT_KEY = "api:error:model:agent:key:%s"

	ACCESS_TOKEN_KEY = "api:baidu:access_token:%s"
	GCP_TOKEN_KEY    = "api:gcp:token:%s"
)

const (
	LOCK_USER_KEY = "api:lock:user:%d"
	LOCK_APP_KEY  = "api:lock:app:%d"
	LOCK_SK_KEY   = "api:lock:sk:%s"
)
