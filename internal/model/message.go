package model

type PubMessage struct {
	Action  string `json:"action,omitempty"`   // 消息动作
	OldData any    `json:"old_data,omitempty"` // 旧数据
	NewData any    `json:"new_data,omitempty"` // 新数据
}

type SubMessage struct {
	Action  string `json:"action,omitempty"`   // 消息动作
	OldData any    `json:"old_data,omitempty"` // 旧数据
	NewData any    `json:"new_data,omitempty"` // 新数据
}
