package model

type User struct {
	Id             string   `json:"id,omitempty"`               // ID
	UserId         int      `json:"user_id,omitempty"`          // 用户ID
	Name           string   `json:"name,omitempty"`             // 姓名
	Avatar         string   `json:"avatar,omitempty"`           // 头像
	Email          string   `json:"email,omitempty"`            // 邮箱
	Phone          string   `json:"phone,omitempty"`            // 手机号
	Quota          int      `json:"quota,omitempty"`            // 剩余额度
	UsedQuota      int      `json:"used_quota,omitempty"`       // 已用额度
	QuotaExpiresAt int64    `json:"quota_expires_at,omitempty"` // 额度过期时间
	Groups         []string `json:"groups,omitempty"`           // 分组权限
	Remark         string   `json:"remark,omitempty"`           // 备注
	Status         int      `json:"status,omitempty"`           // 状态[1:正常, 2:禁用, -1:删除]
	Rid            int      `json:"rid,omitempty"`              // 代理商ID
	CreatedAt      string   `json:"created_at,omitempty"`       // 创建时间
	UpdatedAt      string   `json:"updated_at,omitempty"`       // 更新时间
}

type UserQuota struct {
	UserId       int `json:"user_id,omitempty"` // 用户ID
	CurrentQuota int `json:"quota,omitempty"`   // 当前额度
}
