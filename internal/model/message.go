package model

type SubMessage struct {
	Action  string `json:"action,omitempty"`   // 消息动作[新建:create, 更新:update, 删除:delete, 状态:status]
	OldData any    `json:"old_data,omitempty"` // 旧数据
	NewData any    `json:"new_data,omitempty"` // 新数据
}
