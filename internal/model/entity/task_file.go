package entity

import (
	serrors "github.com/iimeta/fastapi-sdk/v2/errors"
)

type TaskFile struct {
	Id           string            `bson:"_id,omitempty"`            // ID
	TraceId      string            `bson:"trace_id,omitempty"`       // 日志ID
	UserId       int               `bson:"user_id,omitempty"`        // 用户ID
	AppId        int               `bson:"app_id,omitempty"`         // 应用ID
	Model        string            `bson:"model,omitempty"`          // 模型
	Purpose      string            `bson:"purpose,omitempty"`        // 用途[assistants, assistants_output, batch, batch_output, fine-tune, fine-tune-results, vision, user_data]
	FileId       string            `bson:"file_id,omitempty"`        // 文件ID
	FileName     string            `bson:"file_name,omitempty"`      // 文件名
	Bytes        int               `bson:"bytes,omitempty"`          // 文件大小
	Status       string            `bson:"status,omitempty"`         // 状态[uploaded:已上传, processed:已处理, error:已失败, expired:已过期, deleted:已删除]
	ExpiresAt    int64             `bson:"expires_at,omitempty"`     // 过期时间
	FileUrl      string            `bson:"file_url,omitempty"`       // 文件地址
	FilePath     string            `bson:"file_path,omitempty"`      // 文件路径
	Error        *serrors.ApiError `bson:"error,omitempty"`          // 错误信息
	BatchTraceId string            `bson:"batch_trace_id,omitempty"` // 批处理日志ID
	Rid          int               `bson:"rid,omitempty"`            // 代理商ID
	Creator      string            `bson:"creator,omitempty"`        // 创建人
	Updater      string            `bson:"updater,omitempty"`        // 更新人
	CreatedAt    int64             `bson:"created_at,omitempty"`     // 创建时间
	UpdatedAt    int64             `bson:"updated_at,omitempty"`     // 更新时间
}
