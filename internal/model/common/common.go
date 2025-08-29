package common

import (
	sdkm "github.com/iimeta/fastapi-sdk/model"
)

type PresetConfig struct {
	IsSupportSystemRole bool   `bson:"is_support_system_role,omitempty" json:"is_support_system_role,omitempty"` // 是否支持system角色
	SystemRolePrompt    string `bson:"system_role_prompt,omitempty"     json:"system_role_prompt,omitempty"`     // system角色预设提示词
	MinTokens           int    `bson:"min_tokens,omitempty"             json:"min_tokens,omitempty"`             // max_tokens取值的最小值
	MaxTokens           int    `bson:"max_tokens,omitempty"             json:"max_tokens,omitempty"`             // max_tokens取值的最大值
	IsSupportStream     bool   `bson:"is_support_stream,omitempty"      json:"is_support_stream,omitempty"`      // 是否支持流式
}

type TextQuota struct {
	BillingMethod   int     `bson:"billing_method,omitempty"   json:"billing_method,omitempty"`   // 计费方式[1:倍率, 2:固定额度]
	PromptRatio     float64 `bson:"prompt_ratio,omitempty"     json:"prompt_ratio,omitempty"`     // 提示倍率(提问倍率)
	CompletionRatio float64 `bson:"completion_ratio,omitempty" json:"completion_ratio,omitempty"` // 补全倍率(回答倍率)
	CachedRatio     float64 `bson:"cached_ratio,omitempty"     json:"cached_ratio,omitempty"`     // 缓存倍率
	FixedQuota      int     `bson:"fixed_quota,omitempty"      json:"fixed_quota,omitempty"`      // 固定额度
}

type ImageQuota struct {
	BillingMethod    int               `bson:"billing_method,omitempty"    json:"billing_method,omitempty"`    // 计费方式[1:倍率, 2:固定额度]
	GenerationQuotas []GenerationQuota `bson:"generation_quotas,omitempty" json:"generation_quotas,omitempty"` // 生成额度
	TextRatio        float64           `bson:"text_ratio,omitempty"        json:"text_ratio,omitempty"`        // 文本倍率
	InputRatio       float64           `bson:"input_ratio,omitempty"       json:"input_ratio,omitempty"`       // 输入倍率
	OutputRatio      float64           `bson:"output_ratio,omitempty"      json:"output_ratio,omitempty"`      // 输出倍率
	CachedRatio      float64           `bson:"cached_ratio,omitempty"      json:"cached_ratio,omitempty"`      // 缓存倍率
	FixedQuota       int               `bson:"fixed_quota,omitempty"       json:"fixed_quota,omitempty"`       // 固定额度
}

type GenerationQuota struct {
	Quality    string `bson:"quality,omitempty"     json:"quality,omitempty"`     // 质量[high, medium, low, hd, standard]
	Width      int    `bson:"width,omitempty"       json:"width,omitempty"`       // 宽度
	Height     int    `bson:"height,omitempty"      json:"height,omitempty"`      // 高度
	FixedQuota int    `bson:"fixed_quota,omitempty" json:"fixed_quota,omitempty"` // 固定额度
	IsDefault  bool   `bson:"is_default,omitempty"  json:"is_default,omitempty"`  // 是否默认选项
}

type AudioQuota struct {
	BillingMethod   int     `bson:"billing_method,omitempty"   json:"billing_method,omitempty"`   // 计费方式[1:倍率, 2:固定额度]
	PromptRatio     float64 `bson:"prompt_ratio,omitempty"     json:"prompt_ratio,omitempty"`     // 提示倍率(提问倍率)
	CompletionRatio float64 `bson:"completion_ratio,omitempty" json:"completion_ratio,omitempty"` // 补全倍率(回答倍率)
	CachedRatio     float64 `bson:"cached_ratio,omitempty"     json:"cached_ratio,omitempty"`     // 缓存倍率
	FixedQuota      int     `bson:"fixed_quota,omitempty"      json:"fixed_quota,omitempty"`      // 固定额度
}

type MultimodalQuota struct {
	BillingRule  int           `bson:"billing_rule,omitempty"  json:"billing_rule,omitempty"`  // 计费规则[1:按官方, 2:按系统]
	TextQuota    TextQuota     `bson:"text_quota,omitempty"    json:"text_quota,omitempty"`    // 文本额度
	VisionQuotas []VisionQuota `bson:"vision_quotas,omitempty" json:"vision_quotas,omitempty"` // 识图额度
	SearchQuota  int           `bson:"search_quota,omitempty"  json:"search_quota,omitempty"`  // 搜索额度(Google)
	SearchQuotas []SearchQuota `bson:"search_quotas,omitempty" json:"search_quotas,omitempty"` // 搜索额度(OpenAI)
}

type VisionQuota struct {
	Mode       string `bson:"mode,omitempty"        json:"mode,omitempty"`        // 模式[low, high, auto]
	FixedQuota int    `bson:"fixed_quota,omitempty" json:"fixed_quota,omitempty"` // 固定额度
	IsDefault  bool   `bson:"is_default,omitempty"  json:"is_default,omitempty"`  // 是否默认选项
}

type RealtimeQuota struct {
	TextQuota  TextQuota  `bson:"text_quota,omitempty"  json:"text_quota,omitempty"`  // 文本额度
	AudioQuota AudioQuota `bson:"audio_quota,omitempty" json:"audio_quota,omitempty"` // 音频额度
	FixedQuota int        `bson:"fixed_quota,omitempty" json:"fixed_quota,omitempty"` // 固定额度
}

type MultimodalAudioQuota struct {
	TextQuota  TextQuota  `bson:"text_quota,omitempty"  json:"text_quota,omitempty"`  // 文本额度
	AudioQuota AudioQuota `bson:"audio_quota,omitempty" json:"audio_quota,omitempty"` // 音频额度
	FixedQuota int        `bson:"fixed_quota,omitempty" json:"fixed_quota,omitempty"` // 固定额度
}

type MidjourneyQuota struct {
	Name       string `bson:"name,omitempty"        json:"name,omitempty"`        // 名称
	Action     string `bson:"action,omitempty"      json:"action,omitempty"`      // 动作[IMAGINE, UPSCALE, VARIATION, ZOOM, PAN, DESCRIBE, BLEND, SHORTEN, SWAP_FACE]
	Path       string `bson:"path,omitempty"        json:"path,omitempty"`        // 路径
	FixedQuota int    `bson:"fixed_quota,omitempty" json:"fixed_quota,omitempty"` // 固定额度
}

type ForwardConfig struct {
	ForwardRule   int      `bson:"forward_rule,omitempty"   json:"forward_rule,omitempty"`   // 转发规则[1:全部转发, 2:按关键字, 3:内容长度, 4:已用额度]
	MatchRule     []int    `bson:"match_rule,omitempty"     json:"match_rule,omitempty"`     // 转发规则为2时的匹配规则[1:智能匹配, 2:正则匹配]
	TargetModel   string   `bson:"target_model,omitempty"   json:"target_model,omitempty"`   // 转发规则为1和3时的目标模型
	DecisionModel string   `bson:"decision_model,omitempty" json:"decision_model,omitempty"` // 转发规则为2时并且匹配规则为1时的判定模型
	Keywords      []string `bson:"keywords,omitempty"       json:"keywords,omitempty"`       // 转发规则为2时的关键字
	TargetModels  []string `bson:"target_models,omitempty"  json:"target_models,omitempty"`  // 转发规则为2时的目标模型
	ContentLength int      `bson:"content_length,omitempty" json:"content_length,omitempty"` // 转发规则为3时的内容长度
	UsedQuota     int      `bson:"used_quota,omitempty"     json:"used_quota,omitempty"`     // 转发规则为4时的已用额度
}

type FallbackConfig struct {
	ModelAgent     string `bson:"model_agent,omitempty"      json:"model_agent,omitempty"`      // 后备模型代理
	ModelAgentName string `bson:"model_agent_name,omitempty" json:"model_agent_name,omitempty"` // 后备模型代理名称
	Model          string `bson:"model,omitempty"            json:"model,omitempty"`            // 后备模型
	ModelName      string `bson:"model_name,omitempty"       json:"model_name,omitempty"`       // 后备模型名称
}

type Message struct {
	Role         string             `bson:"role,omitempty"          json:"role,omitempty"`    // 角色
	Content      string             `bson:"content,omitempty"       json:"content,omitempty"` // 内容
	Refusal      *string            `bson:"refusal,omitempty"       json:"refusal,omitempty"`
	Name         string             `bson:"name,omitempty"          json:"name,omitempty"`
	FunctionCall *sdkm.FunctionCall `bson:"function_call,omitempty" json:"function_call,omitempty"`
	ToolCalls    any                `bson:"tool_calls,omitempty"    json:"tool_calls,omitempty"`
	ToolCallId   string             `bson:"tool_call_id,omitempty"  json:"tool_call_id,omitempty"`
	Audio        *sdkm.Audio        `bson:"audio,omitempty"         json:"audio,omitempty"`
}

type Retry struct {
	IsRetry    bool   `bson:"is_retry,omitempty"    json:"is_retry,omitempty"`    // 是否重试
	RetryCount int    `bson:"retry_count,omitempty" json:"retry_count,omitempty"` // 重试次数
	ErrMsg     string `bson:"err_msg,omitempty"     json:"err_msg,omitempty"`     // 错误信息
}

type ImageData struct {
	Url           string `bson:"url,omitempty"`
	B64Json       string `bson:"b64_json,omitempty"`
	RevisedPrompt string `bson:"revised_prompt,omitempty"`
}

type SearchQuota struct {
	SearchContextSize string `bson:"search_context_size,omitempty" json:"search_context_size,omitempty"` // 搜索上下文大小[high, medium, low]
	FixedQuota        int    `bson:"fixed_quota,omitempty"         json:"fixed_quota,omitempty"`         // 固定额度
	IsDefault         bool   `bson:"is_default,omitempty"          json:"is_default,omitempty"`          // 是否默认选项
}

type UsageSpend struct {
	TextTokens   int
	ImageTokens  int
	AudioTokens  int
	SearchTokens int
	TotalTokens  int
	Usage        *sdkm.Usage
}
