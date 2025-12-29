package entity

type TaskBatch struct {
	Id           string         `bson:"_id,omitempty"`            // ID
	TraceId      string         `bson:"trace_id,omitempty"`       // 日志ID
	UserId       int            `bson:"user_id,omitempty"`        // 用户ID
	AppId        int            `bson:"app_id,omitempty"`         // 应用ID
	Model        string         `bson:"model,omitempty"`          // 模型
	BatchId      string         `bson:"batch_id,omitempty"`       // 批处理ID
	InputFileId  string         `bson:"input_file_id,omitempty"`  // 输入文件ID
	OutputFileId string         `bson:"output_file_id,omitempty"` // 输出文件ID
	ErrorFileId  string         `bson:"error_file_id,omitempty"`  // 错误文件ID
	Status       string         `bson:"status,omitempty"`         // 状态[validating:验证中, in_progress:进行中, finalizing:定稿中, completed:已完成, cancelling:取消中, cancelled:已取消, failed:已失败, expired:已过期, deleted:已删除]
	InProgressAt int64          `bson:"in_progress_at,omitempty"` // 进行时间
	FinalizingAt int64          `bson:"finalizing_at,omitempty"`  // 定稿时间
	CompletedAt  int64          `bson:"completed_at,omitempty"`   // 完成时间
	ExpiresAt    int64          `bson:"expires_at,omitempty"`     // 过期时间
	CancellingAt int64          `bson:"cancelling_at,omitempty"`  // 取消时间
	CancelledAt  int64          `bson:"cancelled_at,omitempty"`   // 已取消时间
	FailedAt     int64          `bson:"failed_at,omitempty"`      // 失败时间
	ResponseData map[string]any `bson:"response_data,omitempty"`  // 响应数据
	Error        map[string]any `bson:"error,omitempty"`          // 错误信息
	Rid          int            `bson:"rid,omitempty"`            // 代理商ID
	Creator      string         `bson:"creator,omitempty"`        // 创建人
	Updater      string         `bson:"updater,omitempty"`        // 更新人
	CreatedAt    int64          `bson:"created_at,omitempty"`     // 创建时间
	UpdatedAt    int64          `bson:"updated_at,omitempty"`     // 更新时间
}
