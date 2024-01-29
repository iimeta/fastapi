package do

import "github.com/gogf/gf/v2/util/gmeta"

const (
	CHAT_LOG_COLLECTION = "chat_log"
)

type ChatLog struct {
	gmeta.Meta       `collection:"chat_log" bson:"-"`
	TraceId          string    `bson:"trace_id,omitempty"`          // 日志ID
	Corp             string    `bson:"corp,omitempty"`              // 公司[OpenAI;Baidu;Xfyun;Aliyun;Midjourney]
	ModelId          string    `bson:"model_id,omitempty"`          // 模型ID
	Name             string    `bson:"name,omitempty"`              // 模型名称
	Model            string    `bson:"model,omitempty"`             // 模型
	Type             int       `bson:"type,omitempty"`              // 模型类型[1:文生文, 2:文生图, 3:图生文, 4:图生图, 5:文生语音, 6:语音生文]
	BaseUrl          string    `bson:"base_url,omitempty"`          // 默认官方模型地址
	Path             string    `bson:"path,omitempty"`              // 默认官方模型地址路径
	Proxy            string    `bson:"proxy,omitempty"`             // 代理
	Stream           bool      `bson:"stream,omitempty"`            // 是否流式
	Messages         []Message `bson:"messages,omitempty"`          // 完整提示(提问)
	Prompt           string    `bson:"prompt,omitempty"`            // 提示(提问)
	Completion       string    `bson:"completion,omitempty"`        // 补全(回答)
	PromptRatio      float64   `bson:"prompt_ratio,omitempty"`      // 提示倍率(提问倍率)
	CompletionRatio  float64   `bson:"completion_ratio,omitempty"`  // 补全倍率(回答倍率)
	PromptTokens     int       `bson:"prompt_tokens,omitempty"`     // 提示令牌数(提问令牌数)
	CompletionTokens int       `bson:"completion_tokens,omitempty"` // 补全令牌数(回答令牌数)
	TotalTokens      int       `bson:"total_tokens,omitempty"`      // 总令牌数
	ConnTime         int       `bson:"conn_time,omitempty"`         // 连接时间
	Duration         int       `bson:"duration,omitempty"`          // 持续时间
	TotalTime        int       `bson:"total_time,omitempty"`        // 总时间
	Ip               string    `bson:"ip,omitempty"`                // IP
	ErrMsg           string    `bson:"err_msg,omitempty"`           // 错误信息
	Status           int       `bson:"status,omitempty"`            // 状态[1:成功, -1:失败]
	Creator          string    `bson:"creator,omitempty"`           // 创建人
	Updater          string    `bson:"updater,omitempty"`           // 更新人
	CreatedAt        int64     `bson:"created_at,omitempty"`        // 创建时间
	UpdatedAt        int64     `bson:"updated_at,omitempty"`        // 更新时间
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
