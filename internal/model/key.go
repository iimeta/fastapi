package model

type Key struct {
	Id                 string   `json:"id,omitempty"`                   // ID
	ProviderId         string   `json:"provider_id,omitempty"`          // 提供商ID
	Key                string   `json:"key,omitempty"`                  // 密钥
	Weight             int      `json:"weight,omitempty"`               // 权重
	CurrentWeight      int      `json:"current_weight,omitempty"`       // 当前权重
	Models             []string `json:"models,omitempty"`               // 模型
	ModelAgents        []string `json:"model_agents,omitempty"`         // 模型代理
	IsNeverDisable     bool     `json:"is_never_disable,omitempty"`     // 是否永不禁用
	UsedQuota          int      `json:"used_quota,omitempty"`           // 已用额度
	Remark             string   `json:"remark,omitempty"`               // 备注
	Status             int      `json:"status,omitempty"`               // 状态[1:正常, 2:禁用, -1:删除]
	IsAutoDisabled     bool     `json:"is_auto_disabled,omitempty"`     // 是否自动禁用
	AutoDisabledReason string   `json:"auto_disabled_reason,omitempty"` // 自动禁用原因
	Rid                int      `json:"rid,omitempty"`                  // 代理商ID
	Creator            string   `json:"creator,omitempty"`              // 创建人
	Updater            string   `json:"updater,omitempty"`              // 更新人
	CreatedAt          string   `json:"created_at,omitempty"`           // 创建时间
	UpdatedAt          string   `json:"updated_at,omitempty"`           // 更新时间
}
