package model

type ModelAgent struct {
	Id                 string   `json:"id,omitempty"`                   // ID
	Corp               string   `json:"corp,omitempty"`                 // 公司
	Name               string   `json:"name,omitempty"`                 // 模型代理名称
	BaseUrl            string   `json:"base_url,omitempty"`             // 模型代理地址
	Path               string   `json:"path,omitempty"`                 // 模型代理地址路径
	Weight             int      `json:"weight,omitempty"`               // 权重
	CurrentWeight      int      `json:"current_weight,omitempty"`       // 当前权重
	LbStrategy         int      `json:"lb_strategy,omitempty"`          // 密钥负载均衡策略[1:轮询, 2:权重]
	Models             []string `json:"models,omitempty"`               // 绑定模型
	ModelNames         []string `json:"model_names,omitempty"`          // 模型名称
	Key                string   `json:"key,omitempty"`                  // 密钥
	Remark             string   `json:"remark,omitempty"`               // 备注
	Status             int      `json:"status,omitempty"`               // 状态[1:正常, 2:禁用, -1:删除]
	IsAutoDisabled     bool     `json:"is_auto_disabled,omitempty"`     // 是否自动禁用
	AutoDisabledReason string   `json:"auto_disabled_reason,omitempty"` // 自动禁用原因
	Creator            string   `json:"creator,omitempty"`              // 创建人
	Updater            string   `json:"updater,omitempty"`              // 更新人
	CreatedAt          string   `json:"created_at,omitempty"`           // 创建时间
	UpdatedAt          string   `json:"updated_at,omitempty"`           // 更新时间
}
