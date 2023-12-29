package do

import "github.com/gogf/gf/v2/util/gmeta"

const (
	USER_COLLECTION = "user"
)

type User struct {
	gmeta.Meta `collection:"user" bson:"-"`
	UserId     int    `bson:"user_id,omitempty"`    // 用户ID
	Nickname   string `bson:"nickname,omitempty"`   // 昵称
	Avatar     string `bson:"avatar,omitempty"`     // 头像
	Gender     int    `bson:"gender"`               // 性别[0:保密;1:男;2:女]
	Mobile     string `bson:"mobile,omitempty"`     // 手机号
	Email      string `bson:"email,omitempty"`      // 邮箱
	Birthday   string `bson:"birthday"`             // 生日
	Motto      string `bson:"motto"`                // 座右铭
	Status     int    `bson:"status,omitempty"`     // 状态[1:正常;2:禁用;-1:删除]
	CreatedAt  int64  `bson:"created_at,omitempty"` // 注册时间
	UpdatedAt  int64  `bson:"updated_at,omitempty"` // 更新时间
}
