package do

import (
	"github.com/gogf/gf/v2/util/gmeta"
	"github.com/iimeta/fastapi/internal/model/common"
)

const (
	SYS_CONFIG_COLLECTION = "sys_config"
)

type SysConfig struct {
	gmeta.Meta         `collection:"sys_config" bson:"-"`
	Core               *common.Core               `bson:"core,omitempty"`                // 核心
	Http               *common.Http               `bson:"http,omitempty"`                // HTTP
	Email              *common.Email              `bson:"email,omitempty"`               // 邮箱
	Statistics         *common.Statistics         `bson:"statistics,omitempty"`          // 统计
	Base               *common.Base               `bson:"base,omitempty"`                // 基础
	Midjourney         *common.Midjourney         `bson:"midjourney,omitempty"`          // Midjourney
	Log                *common.Log                `bson:"log,omitempty"`                 // 日志
	UserShieldError    *common.UserShieldError    `bson:"user_shield_error,omitempty"`   // 用户屏蔽错误
	AutoDisabledError  *common.AutoDisabledError  `bson:"auto_disabled_error,omitempty"` // 自动禁用错误
	NotRetryError      *common.NotRetryError      `bson:"not_retry_error,omitempty"`     // 不重试错误
	NotShieldError     *common.NotShieldError     `bson:"not_shield_error,omitempty"`    // 不屏蔽错误
	QuotaWarning       *common.QuotaWarning       `bson:"quota_warning,omitempty"`       // 额度预警
	ServiceUnavailable *common.ServiceUnavailable `bson:"service_unavailable,omitempty"` // 暂停服务
	Debug              *common.Debug              `bson:"debug,omitempty"`               // 调试
	Creator            string                     `bson:"creator,omitempty"`             // 创建人
	Updater            string                     `bson:"updater,omitempty"`             // 更新人
	CreatedAt          int64                      `bson:"created_at,omitempty"`          // 创建时间
	UpdatedAt          int64                      `bson:"updated_at,omitempty"`          // 更新时间
}
