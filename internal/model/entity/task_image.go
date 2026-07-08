package entity

import smodel "github.com/iimeta/fastapi-sdk/v2/model"

type TaskImage struct {
	Id             string             `bson:"_id,omitempty"`              // ID
	TraceId        string             `bson:"trace_id,omitempty"`         // 日志ID
	UserId         int                `bson:"user_id,omitempty"`          // 用户ID
	AppId          int                `bson:"app_id,omitempty"`           // 应用ID
	Model          string             `bson:"model,omitempty"`            // 模型
	Action         string             `bson:"action,omitempty"`           // 接口
	ImageId        string             `bson:"image_id,omitempty"`         // 图像ID
	Width          int                `bson:"width,omitempty"`            // 宽度
	Height         int                `bson:"height,omitempty"`           // 高度
	N              int                `bson:"n,omitempty"`                // 生成数量
	Quality        string             `bson:"quality,omitempty"`          // 质量
	Size           string             `bson:"size,omitempty"`             // 尺寸大小
	OutputFormat   string             `bson:"output_format,omitempty"`    // 输出格式
	ResponseFormat string             `bson:"response_format,omitempty"`  // 响应格式
	Prompt         string             `bson:"prompt,omitempty"`           // 提示
	Progress       int                `bson:"progress,omitempty"`         // 进度
	Status         string             `bson:"status,omitempty"`           // 状态[queued:排队中, in_progress:进行中, completed:已完成, failed:已失败, expired:已过期, deleted:已删除]
	CompletedAt    int64              `bson:"completed_at,omitempty"`     // 完成时间
	ExpiresAt      int64              `bson:"expires_at,omitempty"`       // 过期时间
	ImageUrl       string             `bson:"image_url,omitempty"`        // 图像地址
	ImageUrls      []string           `bson:"image_urls,omitempty"`       // 图像地址列表
	FileName       string             `bson:"file_name,omitempty"`        // 文件名
	FileNames      []string           `bson:"file_names,omitempty"`       // 文件名列表
	FilePath       string             `bson:"file_path,omitempty"`        // 文件路径
	FilePaths      []string           `bson:"file_paths,omitempty"`       // 文件路径列表
	InputFilePaths []string           `bson:"input_file_paths,omitempty"` // 输入文件路径列表(异步任务base64转储)
	RequestData    map[string]any     `bson:"request_data,omitempty"`     // 请求数据
	ResponseData   map[string]any     `bson:"response_data,omitempty"`    // 响应数据
	Error          *smodel.ImageError `bson:"error,omitempty"`            // 错误信息
	ModelAgentId   string             `bson:"model_agent_id,omitempty"`   // 模型代理ID
	ModelAgent     *ModelAgent        `bson:"model_agent,omitempty"`      // 模型代理信息
	Rid            int                `bson:"rid,omitempty"`              // 代理商ID
	Creator        string             `bson:"creator,omitempty"`          // 创建人
	Updater        string             `bson:"updater,omitempty"`          // 更新人
	CreatedAt      int64              `bson:"created_at,omitempty"`       // 创建时间
	UpdatedAt      int64              `bson:"updated_at,omitempty"`       // 更新时间
}
