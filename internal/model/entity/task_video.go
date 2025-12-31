package entity

import smodel "github.com/iimeta/fastapi-sdk/v2/model"

type TaskVideo struct {
	Id                 string             `bson:"_id,omitempty"`                   // ID
	TraceId            string             `bson:"trace_id,omitempty"`              // 日志ID
	UserId             int                `bson:"user_id,omitempty"`               // 用户ID
	AppId              int                `bson:"app_id,omitempty"`                // 应用ID
	Model              string             `bson:"model,omitempty"`                 // 模型
	VideoId            string             `bson:"video_id,omitempty"`              // 视频ID
	Width              int                `bson:"width,omitempty"`                 // 宽度
	Height             int                `bson:"height,omitempty"`                // 高度
	Seconds            int                `bson:"seconds,omitempty"`               // 秒数
	Prompt             string             `bson:"prompt,omitempty"`                // 提示
	Progress           int                `bson:"progress,omitempty"`              // 进度
	RemixedFromVideoId string             `bson:"remixed_from_video_id,omitempty"` // 混合ID
	Status             string             `bson:"status,omitempty"`                // 状态[queued:排队中, in_progress:进行中, completed:已完成, failed:已失败, expired:已过期, deleted:已删除]
	CompletedAt        int64              `bson:"completed_at,omitempty"`          // 完成时间
	ExpiresAt          int64              `bson:"expires_at,omitempty"`            // 过期时间
	VideoUrl           string             `bson:"video_url,omitempty"`             // 视频地址
	FileName           string             `bson:"file_name,omitempty"`             // 文件名
	FilePath           string             `bson:"file_path,omitempty"`             // 文件路径
	Error              *smodel.VideoError `bson:"error,omitempty"`                 // 错误信息
	Rid                int                `bson:"rid,omitempty"`                   // 代理商ID
	Creator            string             `bson:"creator,omitempty"`               // 创建人
	Updater            string             `bson:"updater,omitempty"`               // 更新人
	CreatedAt          int64              `bson:"created_at,omitempty"`            // 创建时间
	UpdatedAt          int64              `bson:"updated_at,omitempty"`            // 更新时间
}
