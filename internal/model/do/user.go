package do

import "github.com/gogf/gf/v2/util/gmeta"

const (
	USER_COLLECTION = "user"
)

type User struct {
	gmeta.Meta `collection:"user" bson:"-"`
	UserId     int      `bson:"user_id,omitempty"`    // 用户ID
	Name       string   `bson:"name,omitempty"`       // 姓名
	Avatar     string   `bson:"avatar,omitempty"`     // 头像
	Email      string   `bson:"email,omitempty"`      // 邮箱
	Phone      string   `bson:"phone,omitempty"`      // 手机号
	VipLevel   int      `bson:"vip_level,omitempty"`  // 会员等级
	Quota      int      `bson:"quota,omitempty"`      // 额度
	Models     []string `bson:"models,omitempty"`     // 模型权限
	Remark     string   `bson:"remark,omitempty"`     // 备注
	Status     int      `bson:"status,omitempty"`     // 状态[1:正常, 2:禁用, -1:删除]
	Creator    string   `bson:"creator,omitempty"`    // 创建人
	Updater    string   `bson:"updater,omitempty"`    // 更新人
	CreatedAt  int64    `bson:"created_at,omitempty"` // 创建时间
	UpdatedAt  int64    `bson:"updated_at,omitempty"` // 更新时间
}
