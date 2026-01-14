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

const (
	CHANGE_CHANNEL_CONFIG   = "admin:change:channel:config"
	CHANGE_CHANNEL_RESELLER = "admin:change:channel:reseller"
	CHANGE_CHANNEL_USER     = "admin:change:channel:user"
	CHANGE_CHANNEL_APP      = "admin:change:channel:app"
	CHANGE_CHANNEL_APP_KEY  = "admin:change:channel:app:key"
	CHANGE_CHANNEL_PROVIDER = "admin:change:channel:provider"
	CHANGE_CHANNEL_MODEL    = "admin:change:channel:model"
	CHANGE_CHANNEL_KEY      = "admin:change:channel:key"
	CHANGE_CHANNEL_AGENT    = "admin:change:channel:agent"
	CHANGE_CHANNEL_GROUP    = "admin:change:channel:group"
)

const (
	REFRESH_CHANNEL_API = "admin:refresh:channel:api"
)

const (
	ACTION_CREATE   = "create"
	ACTION_UPDATE   = "update"
	ACTION_DELETE   = "delete"
	ACTION_STATUS   = "status"
	ACTION_MODELS   = "models"
	ACTION_CACHE    = "cache"
	ACTION_REMIX    = "remix"
	ACTION_LIST     = "list"
	ACTION_RETRIEVE = "retrieve"
	ACTION_CONTENT  = "content"
	ACTION_UPLOAD   = "upload"
	ACTION_CANCEL   = "cancel"
)

var RESOLUTION_ASPECT_RATIO = map[string]string{
	"1K1:1": "1024x1024",
	"2K1:1": "2048x2048",
	"4K1:1": "4096x4096",

	"1K2:3": "848x1264",
	"2K2:3": "1696x2528",
	"4K2:3": "3392x5056",

	"1K3:2": "1264x848",
	"2K3:2": "2528x1696",
	"4K3:2": "5056x3392",

	"1K3:4": "896x1200",
	"2K3:4": "1792x2400",
	"4K3:4": "3584x4800",

	"1K4:3": "1200x896",
	"2K4:3": "2400x1792",
	"4K4:3": "4800x3584",

	"1K4:5": "928x1152",
	"2K4:5": "1856x2304",
	"4K4:5": "3712x4608",

	"1K5:4": "1152x928",
	"2K5:4": "2304x1856",
	"4K5:4": "4608x3712",

	"1K9:16": "768x1376",
	"2K9:16": "1536x2752",
	"4K9:16": "3072x5504",

	"1K16:9": "1376x768",
	"2K16:9": "2752x1536",
	"4K16:9": "5504x3072",

	"1K21:9": "1584x672",
	"2K21:9": "3168x1344",
	"4K21:9": "6336x2688",
}
