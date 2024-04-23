package model

type App struct {
	Id           string   `json:"id,omitempty"`             // ID
	AppId        int      `json:"app_id,omitempty"`         // 应用ID
	Name         string   `json:"name,omitempty"`           // 应用名称
	Type         int      `json:"type,omitempty"`           // 应用类型
	Models       []string `json:"models,omitempty"`         // 模型权限
	IsLimitQuota bool     `json:"is_limit_quota,omitempty"` // 是否限制额度
	Quota        int      `json:"quota,omitempty"`          // 剩余额度
	UsedQuota    int      `json:"used_quota,omitempty"`     // 已用额度
	IpWhitelist  []string `json:"ip_whitelist,omitempty"`   // IP白名单
	IpBlacklist  []string `json:"ip_blacklist,omitempty"`   // IP黑名单
	Remark       string   `json:"remark,omitempty"`         // 备注
	Status       int      `json:"status,omitempty"`         // 状态[1:正常, 2:禁用, -1:删除]
	UserId       int      `json:"user_id,omitempty"`        // 用户ID
	Creator      string   `json:"creator,omitempty"`        // 创建人
	Updater      string   `json:"updater,omitempty"`        // 更新人
	CreatedAt    string   `json:"created_at,omitempty"`     // 创建时间
	UpdatedAt    string   `json:"updated_at,omitempty"`     // 更新时间
}
