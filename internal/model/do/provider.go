package do

import "github.com/gogf/gf/v2/util/gmeta"

const (
	PROVIDER_COLLECTION = "provider"
)

type Provider struct {
	gmeta.Meta `collection:"provider" bson:"-"`
	Name       string `bson:"name,omitempty"`       // 名称
	Code       string `bson:"code,omitempty"`       // 代码
	Sort       int    `bson:"sort,omitempty"`       // 排序
	IsPublic   bool   `bson:"is_public,omitempty"`  // 是否公开
	Remark     string `bson:"remark,omitempty"`     // 备注
	Status     int    `bson:"status,omitempty"`     // 状态[1:正常, 2:禁用, -1:删除]
	Creator    string `bson:"creator,omitempty"`    // 创建人
	Updater    string `bson:"updater,omitempty"`    // 更新人
	CreatedAt  int64  `bson:"created_at,omitempty"` // 创建时间
	UpdatedAt  int64  `bson:"updated_at,omitempty"` // 更新时间
}
