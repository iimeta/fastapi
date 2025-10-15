package entity

type Reseller struct {
	Id             string   `bson:"_id,omitempty"`              // ID
	UserId         int      `bson:"user_id,omitempty"`          // 用户ID
	Name           string   `bson:"name,omitempty"`             // 姓名
	Avatar         string   `bson:"avatar,omitempty"`           // 头像
	Email          string   `bson:"email,omitempty"`            // 邮箱
	Phone          string   `bson:"phone,omitempty"`            // 手机号
	Quota          int      `bson:"quota,omitempty"`            // 剩余额度
	UsedQuota      int      `bson:"used_quota,omitempty"`       // 已用额度
	QuotaExpiresAt int64    `bson:"quota_expires_at,omitempty"` // 额度过期时间
	Groups         []string `bson:"groups,omitempty"`           // 分组权限
	Remark         string   `bson:"remark,omitempty"`           // 备注
	Status         int      `bson:"status,omitempty"`           // 状态[1:正常, 2:禁用, -1:删除]
	Creator        string   `bson:"creator,omitempty"`          // 创建人
	Updater        string   `bson:"updater,omitempty"`          // 更新人
	CreatedAt      int64    `bson:"created_at,omitempty"`       // 创建时间
	UpdatedAt      int64    `bson:"updated_at,omitempty"`       // 更新时间
}
