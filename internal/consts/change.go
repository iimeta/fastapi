package consts

import "github.com/iimeta/fastapi/internal/config"

var (
	CHANGE_CHANNEL_USER    = config.Cfg.Core.ChannelPrefix + "admin:change:channel:user"
	CHANGE_CHANNEL_APP     = config.Cfg.Core.ChannelPrefix + "admin:change:channel:app"
	CHANGE_CHANNEL_APP_KEY = config.Cfg.Core.ChannelPrefix + "admin:change:channel:app:key"
	CHANGE_CHANNEL_CORP    = config.Cfg.Core.ChannelPrefix + "admin:change:channel:corp"
	CHANGE_CHANNEL_MODEL   = config.Cfg.Core.ChannelPrefix + "admin:change:channel:model"
	CHANGE_CHANNEL_KEY     = config.Cfg.Core.ChannelPrefix + "admin:change:channel:key"
	CHANGE_CHANNEL_AGENT   = config.Cfg.Core.ChannelPrefix + "admin:change:channel:agent"
)

const (
	ACTION_CREATE = "create"
	ACTION_UPDATE = "update"
	ACTION_DELETE = "delete"
	ACTION_STATUS = "status"
	ACTION_MODELS = "models"
)
