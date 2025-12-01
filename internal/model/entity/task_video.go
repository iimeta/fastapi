package entity

type TaskVideo struct {
	Id        string  `bson:"_id,omitempty"`        // ID
	TraceId   string  `bson:"trace_id,omitempty"`   // 日志ID
	UserId    int     `bson:"user_id,omitempty"`    // 用户ID
	AppId     int     `bson:"app_id,omitempty"`     // 应用ID
	TaskId    string  `bson:"task_id,omitempty"`    // 任务ID
	VideoUrl  string  `bson:"video_url,omitempty"`  // 视频地址
	VideoTime float64 `bson:"video_time,omitempty"` // 视频时长(秒)
	Status    int     `bson:"status,omitempty"`     // 状态
	Rid       int     `bson:"rid,omitempty"`        // 代理商ID
	Creator   string  `bson:"creator,omitempty"`    // 创建人
	Updater   string  `bson:"updater,omitempty"`    // 更新人
	CreatedAt int64   `bson:"created_at,omitempty"` // 创建时间
	UpdatedAt int64   `bson:"updated_at,omitempty"` // 更新时间
}
