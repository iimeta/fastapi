package entity

import (
	"github.com/iimeta/fastapi/internal/model/common"
)

type SysConfig struct {
	Id                 string                     `bson:"_id,omitempty"`                 // ID
	Core               *common.Core               `bson:"core,omitempty"`                // 核心
	Http               *common.Http               `bson:"http,omitempty"`                // HTTP
	Base               *common.Base               `bson:"base,omitempty"`                // 基础
	Midjourney         *common.Midjourney         `bson:"midjourney,omitempty"`          // Midjourney
	Log                *common.Log                `bson:"log,omitempty"`                 // 日志
	AutoDisabledError  *common.AutoDisabledError  `bson:"auto_disabled_error,omitempty"` // 自动禁用错误
	NotRetryError      *common.NotRetryError      `bson:"not_retry_error,omitempty"`     // 不重试错误
	NotShieldError     *common.NotShieldError     `bson:"not_shield_error,omitempty"`    // 不屏蔽错误
	Quota              *common.Quota              `bson:"quota,omitempty"`               // 额度
	VideoTask          *common.VideoTask          `bson:"video_task,omitempty"`          // 视频任务
	FileTask           *common.FileTask           `bson:"file_task,omitempty"`           // 文件任务
	ServiceUnavailable *common.ServiceUnavailable `bson:"service_unavailable,omitempty"` // 暂停服务
	Debug              *common.Debug              `bson:"debug,omitempty"`               // 调试
	Creator            string                     `bson:"creator,omitempty"`             // 创建人
	Updater            string                     `bson:"updater,omitempty"`             // 更新人
	CreatedAt          int64                      `bson:"created_at,omitempty"`          // 创建时间
	UpdatedAt          int64                      `bson:"updated_at,omitempty"`          // 更新时间
}
