package model

type ModelAgent struct {
	Id                   string   `json:"id,omitempty"`                      // ID
	ProviderId           string   `json:"provider_id,omitempty"`             // 提供商ID
	Name                 string   `json:"name,omitempty"`                    // 模型代理名称
	BaseUrl              string   `json:"base_url,omitempty"`                // 模型代理地址
	Path                 string   `json:"path,omitempty"`                    // 模型代理地址路径
	Weight               int      `json:"weight,omitempty"`                  // 权重
	CurrentWeight        int      `json:"current_weight,omitempty"`          // 当前权重
	Models               []string `json:"models,omitempty"`                  // 绑定模型
	IsEnableModelReplace bool     `json:"is_enable_model_replace,omitempty"` // 是否启用模型替换
	ReplaceModels        []string `json:"replace_models,omitempty"`          // 替换模型
	TargetModels         []string `json:"target_models,omitempty"`           // 目标模型
	IsNeverDisable       bool     `json:"is_never_disable,omitempty"`        // 是否永不禁用
	LbStrategy           int      `json:"lb_strategy,omitempty"`             // 密钥负载均衡策略[1:轮询, 2:权重]
	Remark               string   `json:"remark,omitempty"`                  // 备注
	Status               int      `json:"status,omitempty"`                  // 状态[1:正常, 2:禁用, -1:删除]
	IsAutoDisabled       bool     `json:"is_auto_disabled,omitempty"`        // 是否自动禁用
	AutoDisabledReason   string   `json:"auto_disabled_reason,omitempty"`    // 自动禁用原因
	Creator              string   `json:"creator,omitempty"`                 // 创建人
	Updater              string   `json:"updater,omitempty"`                 // 更新人
	CreatedAt            string   `json:"created_at,omitempty"`              // 创建时间
	UpdatedAt            string   `json:"updated_at,omitempty"`              // 更新时间
}
