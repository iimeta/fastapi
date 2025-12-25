package entity

type TaskBatch struct {
	Id           string         `bson:"_id,omitempty"`           // ID
	TraceId      string         `bson:"trace_id,omitempty"`      // 日志ID
	UserId       int            `bson:"user_id,omitempty"`       // 用户ID
	AppId        int            `bson:"app_id,omitempty"`        // 应用ID
	Model        string         `bson:"model,omitempty"`         // 模型
	BatchId      string         `bson:"batch_id,omitempty"`      // 批处理ID
	Status       string         `bson:"status,omitempty"`        // 状态[queued:排队中, in_progress:进行中, completed:已完成, failed:已失败, expired:已过期, deleted:已删除]
	CompletedAt  int64          `bson:"completed_at,omitempty"`  // 完成时间
	ExpiresAt    int64          `bson:"expires_at,omitempty"`    // 过期时间
	ResponseData map[string]any `bson:"response_data,omitempty"` // 响应数据
	Rid          int            `bson:"rid,omitempty"`           // 代理商ID
	Creator      string         `bson:"creator,omitempty"`       // 创建人
	Updater      string         `bson:"updater,omitempty"`       // 更新人
	CreatedAt    int64          `bson:"created_at,omitempty"`    // 创建时间
	UpdatedAt    int64          `bson:"updated_at,omitempty"`    // 更新时间
}
