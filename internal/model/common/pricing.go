package common

import (
	smodel "github.com/iimeta/fastapi-sdk/model"
)

type Pricing struct {
	BillingRule     int                      `bson:"billing_rule,omitempty"      json:"billing_rule,omitempty"`      // 计费规则[1:按官方, 2:按系统]
	BillingMethods  []int                    `bson:"billing_methods,omitempty"   json:"billing_methods,omitempty"`   // 计费方式[1:按Tokens, 2:按次]
	BillingItems    []string                 `bson:"billing_items,omitempty"     json:"billing_items,omitempty"`     // 计费项[text:文本, text_cache:文本缓存, tiered_text:阶梯文本, tiered_text_cache:阶梯文本缓存, image:图像, image_generation:图像生成, image_cache:图像缓存, vision:识图, audio:音频, audio_cache:音频缓存, search:搜索, midjourney:Midjourney, once:一次]
	Text            TextPricing              `bson:"text,omitempty"              json:"text,omitempty"`              // 文本
	TextCache       CachePricing             `bson:"text_cache,omitempty"        json:"text_cache,omitempty"`        // 文本缓存
	TieredText      []TextPricing            `bson:"tiered_text,omitempty"       json:"tiered_text,omitempty"`       // 阶梯文本
	TieredTextCache []CachePricing           `bson:"tiered_text_cache,omitempty" json:"tiered_text_cache,omitempty"` // 阶梯文本缓存
	Image           ImagePricing             `bson:"image,omitempty"             json:"image,omitempty"`             // 图像
	ImageGeneration []ImageGenerationPricing `bson:"image_generation,omitempty"  json:"image_generation,omitempty"`  // 图像生成
	ImageCache      CachePricing             `bson:"image_cache,omitempty"       json:"image_cache,omitempty"`       // 图像缓存
	Vision          []VisionPricing          `bson:"vision,omitempty"            json:"vision,omitempty"`            // 识图
	Audio           AudioPricing             `bson:"audio,omitempty"             json:"audio,omitempty"`             // 音频
	AudioCache      CachePricing             `bson:"audio_cache,omitempty"       json:"audio_cache,omitempty"`       // 音频缓存
	Search          []SearchPricing          `bson:"search,omitempty"            json:"search,omitempty"`            // 搜索
	Midjourney      []MidjourneyPricing      `bson:"midjourney,omitempty"        json:"midjourney,omitempty"`        // Midjourney
	Once            OncePricing              `bson:"once,omitempty"              json:"once,omitempty"`              // 一次
}

type TextPricing struct {
	InputRatio  float64 `bson:"input_ratio,omitempty"  json:"input_ratio,omitempty"`  // 输入倍率
	OutputRatio float64 `bson:"output_ratio,omitempty" json:"output_ratio,omitempty"` // 输出倍率
	Mode        string  `bson:"mode,omitempty"         json:"mode,omitempty"`         // 模式[all:全部, thinking:思考, non_thinking:非思考]
	Gt          int     `bson:"gt,omitempty"           json:"gt,omitempty"`           // 大于, 单位: k
	Lte         int     `bson:"lte,omitempty"          json:"lte,omitempty"`          // 小于等于, 单位: k
}

type CachePricing struct {
	ReadRatio  float64 `bson:"read_ratio,omitempty"  json:"read_ratio,omitempty"`  // 读取/命中倍率
	WriteRatio float64 `bson:"write_ratio,omitempty" json:"write_ratio,omitempty"` // 写入倍率
	Mode       string  `bson:"mode,omitempty"        json:"mode,omitempty"`        // 模式[all:全部, thinking:思考, non_thinking:非思考]
	Gt         int     `bson:"gt,omitempty"          json:"gt,omitempty"`          // 大于, 单位: k
	Lte        int     `bson:"lte,omitempty"         json:"lte,omitempty"`         // 小于等于, 单位: k
}

type ImagePricing struct {
	InputRatio  float64 `bson:"input_ratio,omitempty"  json:"input_ratio,omitempty"`  // 输入倍率
	OutputRatio float64 `bson:"output_ratio,omitempty" json:"output_ratio,omitempty"` // 输出倍率
}

type ImageGenerationPricing struct {
	Quality   string  `bson:"quality,omitempty"    json:"quality,omitempty"`    // 质量[high, medium, low, hd, standard]
	Width     int     `bson:"width,omitempty"      json:"width,omitempty"`      // 宽度
	Height    int     `bson:"height,omitempty"     json:"height,omitempty"`     // 高度
	OnceRatio float64 `bson:"once_ratio,omitempty" json:"once_ratio,omitempty"` // 一次倍率
	IsDefault bool    `bson:"is_default,omitempty" json:"is_default,omitempty"` // 是否默认选项
}

type VisionPricing struct {
	Mode      string  `bson:"mode,omitempty"       json:"mode,omitempty"`       // 模式[low, high, auto]
	OnceRatio float64 `bson:"once_ratio,omitempty" json:"once_ratio,omitempty"` // 一次倍率
	IsDefault bool    `bson:"is_default,omitempty" json:"is_default,omitempty"` // 是否默认选项
}

type AudioPricing struct {
	InputRatio  float64 `bson:"input_ratio,omitempty"  json:"input_ratio,omitempty"`  // 输入倍率
	OutputRatio float64 `bson:"output_ratio,omitempty" json:"output_ratio,omitempty"` // 输出倍率
}

type SearchPricing struct {
	ContextSize string  `bson:"context_size,omitempty" json:"context_size,omitempty"` // 上下文大小[high, medium, low]
	OnceRatio   float64 `bson:"once_ratio,omitempty"   json:"once_ratio,omitempty"`   // 一次倍率
	IsDefault   bool    `bson:"is_default,omitempty"   json:"is_default,omitempty"`   // 是否默认选项
}

type MidjourneyPricing struct {
	Name      string  `bson:"name,omitempty"       json:"name,omitempty"`       // 名称
	Action    string  `bson:"action,omitempty"     json:"action,omitempty"`     // 动作[IMAGINE, UPSCALE, VARIATION, ZOOM, PAN, DESCRIBE, BLEND, SHORTEN, SWAP_FACE]
	Path      string  `bson:"path,omitempty"       json:"path,omitempty"`       // 路径
	OnceRatio float64 `bson:"once_ratio,omitempty" json:"once_ratio,omitempty"` // 一次倍率
}

type OncePricing struct {
	OnceRatio float64 `bson:"once_ratio,omitempty" json:"once_ratio,omitempty"` // 一次倍率
}

type BillingData struct {
	Path                   string
	ChatCompletionRequest  smodel.ChatCompletionRequest
	ImageGenerationRequest smodel.ImageGenerationRequest
	ImageEditRequest       smodel.ImageEditRequest
	Completion             string
	AudioInput             string
	AudioMinute            float64
	EmbeddingRequest       smodel.EmbeddingRequest
	ModerationRequest      smodel.ModerationRequest
	Usage                  *smodel.Usage
}

type Spend struct {
	BillingRule            int                    `bson:"billing_rule,omitempty"              json:"billing_rule,omitempty"`              // 计费规则[1:按官方, 2:按系统]
	BillingMethods         []int                  `bson:"billing_methods,omitempty"           json:"billing_methods,omitempty"`           // 计费方式[1:按Tokens, 2:按次]
	BillingItems           []string               `bson:"billing_items,omitempty"             json:"billing_items,omitempty"`             // 计费项[text:文本, text_cache:文本缓存, tiered_text:阶梯文本, tiered_text_cache:阶梯文本缓存, image:图像, image_generation:图像生成, image_cache:图像缓存, vision:识图, audio:音频, audio_cache:音频缓存, search:搜索, midjourney:Midjourney, once:一次]
	TextPricing            TextPricing            `bson:"text_pricing,omitempty"              json:"text_pricing,omitempty"`              // 文本定价
	TextTokens             int                    `bson:"text_tokens,omitempty"               json:"text_tokens,omitempty"`               // 文本花费
	TextCachePricing       CachePricing           `bson:"text_cache_pricing,omitempty"        json:"text_cache_pricing,omitempty"`        // 文本缓存定价
	TextCacheTokens        int                    `bson:"text_cache_tokens,omitempty"         json:"text_cache_tokens,omitempty"`         // 文本缓存花费
	TieredTextPricing      TextPricing            `bson:"tiered_text_pricing,omitempty"       json:"tiered_text_pricing,omitempty"`       // 阶梯文本定价
	TieredTextTokens       int                    `bson:"tiered_text_tokens,omitempty"        json:"tiered_text_tokens,omitempty"`        // 阶梯文本花费
	TieredTextCachePricing CachePricing           `bson:"tiered_text_cache_pricing,omitempty" json:"tiered_text_cache_pricing,omitempty"` // 阶梯文本缓存定价
	TieredTextCacheTokens  int                    `bson:"tiered_text_cache_tokens,omitempty"  json:"tiered_text_cache_tokens,omitempty"`  // 阶梯文本缓存花费
	ImagePricing           ImagePricing           `bson:"image_pricing,omitempty"             json:"image_pricing,omitempty"`             // 图像定价
	ImageTokens            int                    `bson:"image_tokens,omitempty"              json:"image_tokens,omitempty"`              // 图像花费
	ImageGenerationPricing ImageGenerationPricing `bson:"image_generation_pricing,omitempty"  json:"image_generation_pricing,omitempty"`  // 图像生成定价
	ImageGenerationTokens  int                    `bson:"image_generation_tokens,omitempty"   json:"image_generation_tokens,omitempty"`   // 图像生成花费
	ImageCachePricing      CachePricing           `bson:"image_cache_pricing,omitempty"       json:"image_cache_pricing,omitempty"`       // 图像缓存定价
	ImageCacheTokens       int                    `bson:"image_cache_tokens,omitempty"        json:"image_cache_tokens,omitempty"`        // 图像缓存花费
	VisionPricing          VisionPricing          `bson:"vision_pricing,omitempty"            json:"vision_pricing,omitempty"`            // 识图定价
	VisionTokens           int                    `bson:"vision_tokens,omitempty"             json:"vision_tokens,omitempty"`             // 识图花费
	AudioPricing           AudioPricing           `bson:"audio_pricing,omitempty"             json:"audio_pricing,omitempty"`             // 音频定价
	AudioTokens            int                    `bson:"audio_tokens,omitempty"              json:"audio_tokens,omitempty"`              // 音频花费
	AudioCachePricing      CachePricing           `bson:"audio_cache_pricing,omitempty"       json:"audio_cache_pricing,omitempty"`       // 音频缓存定价
	AudioCacheTokens       int                    `bson:"audio_cache_tokens,omitempty"        json:"audio_cache_tokens,omitempty"`        // 音频缓存花费
	SearchPricing          SearchPricing          `bson:"search_pricing,omitempty"            json:"search_pricing,omitempty"`            // 搜索定价
	SearchTokens           int                    `bson:"search_tokens,omitempty"             json:"search_tokens,omitempty"`             // 搜索花费
	MidjourneyPricing      MidjourneyPricing      `bson:"midjourney_pricing,omitempty"        json:"midjourney_pricing,omitempty"`        // Midjourney定价
	MidjourneyTokens       int                    `bson:"midjourney_tokens,omitempty"         json:"midjourney_tokens,omitempty"`         // Midjourney花费
	OncePricing            OncePricing            `bson:"once_pricing,omitempty"              json:"once_pricing,omitempty"`              // 一次定价
	OnceTokens             int                    `bson:"once_tokens,omitempty"               json:"once_tokens,omitempty"`               // 一次花费
	GroupId                string                 `bson:"group_id,omitempty"                  json:"group_id,omitempty"`                  // 分组ID
	GroupName              string                 `bson:"group_name,omitempty"                json:"group_name,omitempty"`                // 分组名称
	GroupDiscount          float64                `bson:"group_discount,omitempty"            json:"group_discount,omitempty"`            // 分组折扣
	TotalTokens            int                    `bson:"total_tokens,omitempty"              json:"total_tokens,omitempty"`              // 总花费
}
