package entity

type Key struct {
	Id                 string   `bson:"_id,omitempty"`                  // ID
	ProviderId         string   `bson:"provider_id,omitempty"`          // 提供商ID
	Key                string   `bson:"key,omitempty"`                  // 密钥
	Weight             int      `bson:"weight,omitempty"`               // 权重
	Models             []string `bson:"models,omitempty"`               // 模型
	ModelAgents        []string `bson:"model_agents,omitempty"`         // 模型代理
	IsAgentsOnly       bool     `bson:"is_agents_only,omitempty"`       // 是否代理专用
	IsNeverDisable     bool     `bson:"is_never_disable,omitempty"`     // 是否永不禁用
	UsedQuota          int      `bson:"used_quota,omitempty"`           // 已用额度
	Remark             string   `bson:"remark,omitempty"`               // 备注
	Status             int      `bson:"status,omitempty"`               // 状态[1:正常, 2:禁用, -1:删除]
	IsAutoDisabled     bool     `bson:"is_auto_disabled,omitempty"`     // 是否自动禁用
	AutoDisabledReason string   `bson:"auto_disabled_reason,omitempty"` // 自动禁用原因
	Rid                int      `bson:"rid,omitempty"`                  // 代理商ID
	Creator            string   `bson:"creator,omitempty"`              // 创建人
	Updater            string   `bson:"updater,omitempty"`              // 更新人
	CreatedAt          int64    `bson:"created_at,omitempty"`           // 创建时间
	UpdatedAt          int64    `bson:"updated_at,omitempty"`           // 更新时间
}
