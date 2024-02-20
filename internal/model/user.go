package model

type User struct {
	Id        string `json:"id,omitempty"`         // ID
	UserId    int    `json:"user_id,omitempty"`    // 用户ID
	Name      string `json:"name,omitempty"`       // 姓名
	Avatar    string `json:"avatar,omitempty"`     // 用户头像地址
	Gender    int    `json:"gender,omitempty"`     // 用户性别  0:未知  1:男   2:女
	Phone     string `json:"phone,omitempty"`      // 手机号
	Email     string `json:"email,omitempty"`      // 用户邮箱
	Quota     int    `json:"quota"`                // 额度
	Remark    string `json:"remark,omitempty"`     // 备注
	Status    int    `json:"status,omitempty"`     // 状态[1:正常, 2:禁用, -1:删除]
	CreatedAt string `json:"created_at,omitempty"` // 创建时间
	UpdatedAt string `json:"updated_at,omitempty"` // 更新时间
}
