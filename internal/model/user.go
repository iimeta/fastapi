package model

type User struct {
	Id        string `json:"id,omitempty"`         // ID
	UserId    int    `json:"user_id,omitempty"`    // 用户ID
	Nickname  string `json:"nickname,omitempty"`   // 用户昵称
	Avatar    string `json:"avatar,omitempty"`     // 用户头像地址
	Gender    int    `json:"gender,omitempty"`     // 用户性别  0:未知  1:男   2:女
	Mobile    string `json:"mobile,omitempty"`     // 手机号
	Email     string `json:"email,omitempty"`      // 用户邮箱
	Quota     int    `json:"quota,omitempty"`      // 额度
	Remark    string `json:"remark,omitempty"`     // 备注
	CreatedAt int64  `json:"created_at,omitempty"` // 创建时间
	UpdatedAt int64  `json:"updated_at,omitempty"` // 更新时间
}
