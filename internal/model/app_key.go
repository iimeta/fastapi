package model

type AppKey struct {
	Id                  string   `json:"id,omitempty"`                 // ID
	UserId              int      `json:"user_id,omitempty"`            // 用户ID
	AppId               int      `json:"app_id,omitempty"`             // 应用ID
	Key                 string   `json:"key,omitempty"`                // 密钥
	BillingMethods      []int    `json:"billing_methods,omitempty"`    // 计费方式[1:按Tokens, 2:按次]
	Models              []string `json:"models,omitempty"`             // 模型
	IsLimitQuota        bool     `json:"is_limit_quota"`               // 是否限制额度
	Quota               int      `json:"quota,omitempty"`              // 剩余额度
	UsedQuota           int      `json:"used_quota,omitempty"`         // 已用额度
	QuotaExpiresRule    int      `json:"quota_expires_rule,omitempty"` // 额度过期规则[1:固定, 2:时长]
	QuotaExpiresAt      int64    `json:"quota_expires_at,omitempty"`   // 额度过期时间
	QuotaExpiresMinutes int64    `json:"quota_expires_minutes"`        // 额度过期分钟数
	IsBindGroup         bool     `json:"is_bind_group,omitempty"`      // 是否绑定分组
	Group               string   `json:"group,omitempty"`              // 绑定分组
	IpWhitelist         []string `json:"ip_whitelist,omitempty"`       // IP白名单
	IpBlacklist         []string `json:"ip_blacklist,omitempty"`       // IP黑名单
	Remark              string   `json:"remark,omitempty"`             // 备注
	Status              int      `json:"status,omitempty"`             // 状态[1:正常, 2:禁用, -1:删除]
	Rid                 int      `json:"rid,omitempty"`                // 代理商ID
	Creator             string   `json:"creator,omitempty"`            // 创建人
	Updater             string   `json:"updater,omitempty"`            // 更新人
	CreatedAt           string   `json:"created_at,omitempty"`         // 创建时间
	UpdatedAt           string   `json:"updated_at,omitempty"`         // 更新时间
}
