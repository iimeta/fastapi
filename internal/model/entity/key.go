package entity

type Key struct {
	Id           string   `bson:"_id,omitempty"`            // ID
	UserId       int      `bson:"user_id,omitempty"`        // 用户ID
	AppId        int      `bson:"app_id,omitempty"`         // 应用ID
	Corp         string   `bson:"corp,omitempty"`           // 公司[OpenAI;Baidu;Xfyun;Aliyun;ZhipuAI;Google;DeepSeek;Midjourney]
	Key          string   `bson:"key,omitempty"`            // 密钥
	Type         int      `bson:"type,omitempty"`           // 密钥类型[1:应用, 2:模型]
	Models       []string `bson:"models,omitempty"`         // 模型
	ModelAgents  []string `bson:"model_agents,omitempty"`   // 模型代理
	IsAgentsOnly bool     `bson:"is_agents_only,omitempty"` // 是否代理专用
	IsLimitQuota bool     `bson:"is_limit_quota,omitempty"` // 是否限制额度
	Quota        int      `bson:"quota,omitempty"`          // 剩余额度
	UsedQuota    int      `bson:"used_quota,omitempty"`     // 已用额度
	RPM          int      `bson:"rpm,omitempty"`            // 每分钟请求数
	RPD          int      `bson:"rpd,omitempty"`            // 每天的请求数
	IpWhitelist  []string `bson:"ip_whitelist,omitempty"`   // IP白名单
	IpBlacklist  []string `bson:"ip_blacklist,omitempty"`   // IP黑名单
	Remark       string   `bson:"remark,omitempty"`         // 备注
	Status       int      `bson:"status,omitempty"`         // 状态[1:正常, 2:禁用, -1:删除]
	Creator      string   `bson:"creator,omitempty"`        // 创建人
	Updater      string   `bson:"updater,omitempty"`        // 更新人
	CreatedAt    int64    `bson:"created_at,omitempty"`     // 创建时间
	UpdatedAt    int64    `bson:"updated_at,omitempty"`     // 更新时间
}
