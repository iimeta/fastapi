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
	InputGt     int     `bson:"input_gt,omitempty"     json:"input_gt,omitempty"`     // 输入大于, 单位: k
	InputLte    int     `bson:"input_lte,omitempty"    json:"input_lte,omitempty"`    // 输入小于等于, 单位: k
}

type CachePricing struct {
	ReadRatio  float64 `bson:"read_ratio,omitempty"  json:"read_ratio,omitempty"`  // 读取/命中倍率
	WriteRatio float64 `bson:"write_ratio,omitempty" json:"write_ratio,omitempty"` // 写入倍率
	Mode       string  `bson:"mode,omitempty"        json:"mode,omitempty"`        // 模式[all:全部, thinking:思考, non_thinking:非思考]
	InputGt    int     `bson:"input_gt,omitempty"    json:"input_gt,omitempty"`    // 输入大于, 单位: k
	InputLte   int     `bson:"input_lte,omitempty"   json:"input_lte,omitempty"`   // 输入小于等于, 单位: k
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

type UsageSpend struct {
	ChatCompletionRequest smodel.ChatCompletionRequest
	Completion            string
	Usage                 *smodel.Usage
}

type UsageSpendTokens struct {
	TextTokens            int // 文本
	TextCacheTokens       int // 文本缓存
	TieredTextTokens      int // 阶梯文本
	TieredTextCacheTokens int // 阶梯文本缓存
	ImageTokens           int // 图像
	ImageGenerationTokens int // 图像生成
	ImageCacheTokens      int // 图像缓存
	VisionTokens          int // 识图
	AudioTokens           int // 音频
	AudioCacheTokens      int // 音频缓存
	SearchTokens          int // 搜索
	MidjourneyTokens      int // Midjourney
	OnceTokens            int // 一次
	TotalTokens           int // 总
}
