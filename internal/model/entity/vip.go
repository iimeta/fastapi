package entity

const (
	VIP_COLLECTION = "vip"
)

type Vip struct {
	Id          string   `bson:"_id,omitempty"`          // ID
	Level       int      `bson:"level,omitempty"`        // 等级
	Name        string   `bson:"name,omitempty"`         // 名称
	Models      []string `bson:"models,omitempty"`       // 模型权限
	FreeTokens  int      `bson:"free_tokens,omitempty"`  // 免费额度
	MinuteLimit int      `bson:"minute_limit,omitempty"` // 分钟限额
	DailyLimit  int      `bson:"daily_limit,omitempty"`  // 每日限额
	Rule        Rule     `bson:"rule,omitempty"`         // 规则
	Remark      string   `bson:"remark,omitempty"`       // 备注
	Status      int      `bson:"status,omitempty"`       // 状态[1:正常;2:下线;-1:删除]
	CreatedAt   int64    `bson:"created_at,omitempty"`   // 创建时间
	UpdatedAt   int64    `bson:"updated_at,omitempty"`   // 更新时间
}

type InviteRecord struct {
	Id        string `bson:"_id,omitempty"`        // ID
	Nickname  string `bson:"nickname,omitempty"`   // 用户昵称
	Email     string `bson:"email,omitempty"`      // 用户邮箱
	CreatedAt int64  `bson:"created_at,omitempty"` // 注册时间
	Inviter   int    `bson:"inviter,omitempty"`    // 邀请人
}

type Rule struct {
	OnlineTime    int `bson:"online_time,omitempty"`     // 在线时间
	RegDays       int `bson:"reg_days,omitempty"`        // 注册天数
	InviteNum     int `bson:"invite_num,omitempty"`      // 邀请人数
	InviteRegDays int `bson:"invite_reg_days,omitempty"` // 邀请注册天数
}
