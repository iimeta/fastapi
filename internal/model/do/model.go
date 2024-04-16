package do

import "github.com/gogf/gf/v2/util/gmeta"

const (
	MODEL_COLLECTION = "model"
)

type Model struct {
	gmeta.Meta         `collection:"model" bson:"-"`
	Corp               string   `bson:"corp,omitempty"`             // 公司[OpenAI;Baidu;Xfyun;Aliyun;Midjourney]
	Name               string   `bson:"name,omitempty"`             // 模型名称
	Model              string   `bson:"model,omitempty"`            // 模型
	Type               int      `bson:"type,omitempty"`             // 模型类型[1:文生文, 2:文生图, 3:图生文, 4:图生图, 5:文生语音, 6:语音生文]
	BaseUrl            string   `bson:"base_url,omitempty"`         // 模型地址
	Path               string   `bson:"path,omitempty"`             // 模型路径
	Prompt             string   `bson:"prompt,omitempty"`           // 预设提示词
	BillingMethod      int      `bson:"billing_method,omitempty"`   // 计费方式[1:倍率, 2:固定额度]
	PromptRatio        float64  `bson:"prompt_ratio,omitempty"`     // 提示倍率(提问倍率)
	CompletionRatio    float64  `bson:"completion_ratio,omitempty"` // 补全倍率(回答倍率)
	FixedQuota         int      `bson:"fixed_quota,omitempty"`      // 固定额度
	DataFormat         int      `bson:"data_format,omitempty"`      // 数据格式[1:统一格式, 2:官方格式]
	IsPublic           bool     `bson:"is_public"`                  // 是否公开
	IsEnableModelAgent bool     `bson:"is_enable_model_agent"`      // 是否启用模型代理
	ModelAgents        []string `bson:"model_agents,omitempty"`     // 模型代理
	Remark             string   `bson:"remark"`                     // 备注
	Status             int      `bson:"status,omitempty"`           // 状态[1:正常, 2:禁用, -1:删除]
	Creator            string   `bson:"creator,omitempty"`          // 创建人
	Updater            string   `bson:"updater,omitempty"`          // 更新人
	CreatedAt          int64    `bson:"created_at,omitempty"`       // 创建时间
	UpdatedAt          int64    `bson:"updated_at,omitempty"`       // 更新时间
}
