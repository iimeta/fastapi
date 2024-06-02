package model

type Corp struct {
	Id        string `json:"id,omitempty"`         // ID
	Name      string `json:"name,omitempty"`       // 名称
	Code      string `json:"code,omitempty"`       // 代码
	Sort      int    `json:"sort,omitempty"`       // 排序
	IsPublic  bool   `json:"is_public,omitempty"`  // 是否公开
	Remark    string `json:"remark,omitempty"`     // 备注
	Status    int    `json:"status,omitempty"`     // 状态[1:正常, 2:禁用, -1:删除]
	Creator   string `json:"creator,omitempty"`    // 创建人
	Updater   string `json:"updater,omitempty"`    // 更新人
	CreatedAt string `json:"created_at,omitempty"` // 创建时间
	UpdatedAt string `json:"updated_at,omitempty"` // 更新时间
}
