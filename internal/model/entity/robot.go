package entity

type Robot struct {
	Id        string `bson:"_id,omitempty"`        // 机器人ID
	UserId    int    `bson:"user_id,omitempty"`    // 绑定的用户ID
	IsTalk    int    `bson:"is_talk,omitempty"`    // 是否可发送消息[0:否;1:是;]
	Type      int    `bson:"type,omitempty"`       // 机器人类型
	Status    int    `bson:"status,omitempty"`     // 状态[-1:已删除;0:正常;1:已禁用;]
	Corp      string `bson:"corp,omitempty"`       // 公司
	Model     string `bson:"model,omitempty"`      // 模型
	ModelType string `bson:"model_type,omitempty"` // 模型类型, 文生文: text, 画图: image
	Role      string `bson:"role,omitempty"`       // 角色
	Prompt    string `bson:"prompt,omitempty"`     // 提示
	Proxy     string `bson:"proxy,omitempty"`      // 代理
	Key       string `bson:"key,omitempty"`        // 密钥
	CreatedAt int64  `bson:"created_at,omitempty"` // 创建时间
	UpdatedAt int64  `bson:"updated_at,omitempty"` // 更新时间
}
