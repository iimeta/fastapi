package common

import (
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
)

type Pricing struct {
	BillingRule     int                       `bson:"billing_rule,omitempty"      json:"billing_rule,omitempty"`      // 计费规则[1:按官方, 2:按系统]
	BillingMethods  []int                     `bson:"billing_methods,omitempty"   json:"billing_methods,omitempty"`   // 计费方式[1:按Tokens, 2:按次]
	BillingItems    []string                  `bson:"billing_items,omitempty"     json:"billing_items,omitempty"`     // 计费项[text:文本, text_cache:文本缓存, tiered_text:阶梯文本, tiered_text_cache:阶梯文本缓存, image:图像, image_generation:图像生成, image_cache:图像缓存, vision:识图, audio:音频, audio_cache:音频缓存, video:视频, video_generation:视频生成, video_cache:视频缓存, search:搜索, midjourney:Midjourney, once:一次]
	Text            []*TextPricing            `bson:"text,omitempty"              json:"text,omitempty"`              // 文本
	TextCache       []*CachePricing           `bson:"text_cache,omitempty"        json:"text_cache,omitempty"`        // 文本缓存
	TieredText      []*TextPricing            `bson:"tiered_text,omitempty"       json:"tiered_text,omitempty"`       // 阶梯文本
	TieredTextCache []*CachePricing           `bson:"tiered_text_cache,omitempty" json:"tiered_text_cache,omitempty"` // 阶梯文本缓存
	Image           *ImagePricing             `bson:"image,omitempty"             json:"image,omitempty"`             // 图像
	ImageGeneration []*ImageGenerationPricing `bson:"image_generation,omitempty"  json:"image_generation,omitempty"`  // 图像生成
	ImageCache      *CachePricing             `bson:"image_cache,omitempty"       json:"image_cache,omitempty"`       // 图像缓存
	Vision          []*VisionPricing          `bson:"vision,omitempty"            json:"vision,omitempty"`            // 识图
	Audio           *AudioPricing             `bson:"audio,omitempty"             json:"audio,omitempty"`             // 音频
	AudioCache      *CachePricing             `bson:"audio_cache,omitempty"       json:"audio_cache,omitempty"`       // 音频缓存
	Video           *VideoPricing             `bson:"video,omitempty"             json:"video,omitempty"`             // 视频
	VideoGeneration []*VideoGenerationPricing `bson:"video_generation,omitempty"  json:"video_generation,omitempty"`  // 视频生成
	VideoCache      *CachePricing             `bson:"video_cache,omitempty"       json:"video_cache,omitempty"`       // 视频缓存
	Search          []*SearchPricing          `bson:"search,omitempty"            json:"search,omitempty"`            // 搜索
	Midjourney      []*MidjourneyPricing      `bson:"midjourney,omitempty"        json:"midjourney,omitempty"`        // Midjourney
	Once            *OncePricing              `bson:"once,omitempty"              json:"once,omitempty"`              // 一次
}

type TextPricing struct {
	ServiceTier    string  `bson:"service_tier,omitempty"    json:"service_tier,omitempty"`    // 服务层[all:全部, default:默认, priority:优先, flex:弹性]
	InputRatio     float64 `bson:"input_ratio,omitempty"     json:"input_ratio,omitempty"`     // 输入倍率
	OutputRatio    float64 `bson:"output_ratio,omitempty"    json:"output_ratio,omitempty"`    // 输出倍率
	ReasoningRatio float64 `bson:"reasoning_ratio,omitempty" json:"reasoning_ratio,omitempty"` // 思考倍率
	Mode           string  `bson:"mode,omitempty"            json:"mode,omitempty"`            // 模式[all:全部, thinking:思考, non_thinking:非思考]
	Gt             int     `bson:"gt,omitempty"              json:"gt,omitempty"`              // 大于, 单位: k
	Lte            int     `bson:"lte,omitempty"             json:"lte,omitempty"`             // 小于等于, 单位: k
}

type CachePricing struct {
	ServiceTier string  `bson:"service_tier,omitempty" json:"service_tier,omitempty"` // 服务层[all:全部, default:默认, priority:优先, flex:弹性]
	ReadRatio   float64 `bson:"read_ratio,omitempty"   json:"read_ratio,omitempty"`   // 读取/命中倍率
	WriteRatio  float64 `bson:"write_ratio,omitempty"  json:"write_ratio,omitempty"`  // 写入倍率
	Mode        string  `bson:"mode,omitempty"         json:"mode,omitempty"`         // 模式[all:全部, thinking:思考, non_thinking:非思考]
	Gt          int     `bson:"gt,omitempty"           json:"gt,omitempty"`           // 大于, 单位: k
	Lte         int     `bson:"lte,omitempty"          json:"lte,omitempty"`          // 小于等于, 单位: k
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

type VideoPricing struct {
	InputRatio  float64 `bson:"input_ratio"  json:"input_ratio,omitempty"`  // 输入倍率
	OutputRatio float64 `bson:"output_ratio" json:"output_ratio,omitempty"` // 输出倍率
}

type VideoGenerationPricing struct {
	Width     int     `bson:"width,omitempty"      json:"width,omitempty"`      // 宽度
	Height    int     `bson:"height,omitempty"     json:"height,omitempty"`     // 高度
	OnceRatio float64 `bson:"once_ratio"           json:"once_ratio,omitempty"` // 一次倍率
	IsDefault bool    `bson:"is_default,omitempty" json:"is_default,omitempty"` // 是否默认选项
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
	ServiceTier            string
	AudioInput             string
	AudioMinute            float64
	EmbeddingRequest       smodel.EmbeddingRequest
	ModerationRequest      smodel.ModerationRequest
	Seconds                int
	Size                   string
	Usage                  *smodel.Usage
	IsAborted              bool
}

type Spend struct {
	BillingRule      int                   `bson:"billing_rule,omitempty"       json:"billing_rule,omitempty"`       // 计费规则[1:按官方, 2:按系统]
	BillingMethods   []int                 `bson:"billing_methods,omitempty"    json:"billing_methods,omitempty"`    // 计费方式[1:按Tokens, 2:按次]
	BillingItems     []string              `bson:"billing_items,omitempty"      json:"billing_items,omitempty"`      // 计费项[text:文本, text_cache:文本缓存, tiered_text:阶梯文本, tiered_text_cache:阶梯文本缓存, image:图像, image_generation:图像生成, image_cache:图像缓存, vision:识图, audio:音频, audio_cache:音频缓存, video:视频, video_generation:视频生成, video_cache:视频缓存, search:搜索, midjourney:Midjourney, once:一次]
	Text             *TextSpend            `bson:"text,omitempty"               json:"text,omitempty"`               // 文本
	TextCache        *CacheSpend           `bson:"text_cache,omitempty"         json:"text_cache,omitempty"`         // 文本缓存
	TieredText       *TextSpend            `bson:"tiered_text,omitempty"        json:"tiered_text,omitempty"`        // 阶梯文本
	TieredTextCache  *CacheSpend           `bson:"tiered_text_cache,omitempty"  json:"tiered_text_cache,omitempty"`  // 阶梯文本缓存
	Image            *ImageSpend           `bson:"image,omitempty"              json:"image,omitempty"`              // 图像
	ImageGeneration  *ImageGenerationSpend `bson:"image_generation,omitempty"   json:"image_generation,omitempty"`   // 图像生成
	ImageCache       *CacheSpend           `bson:"image_cache,omitempty"        json:"image_cache,omitempty"`        // 图像缓存
	Vision           *VisionSpend          `bson:"vision,omitempty"             json:"vision,omitempty"`             // 识图
	Audio            *AudioSpend           `bson:"audio,omitempty"              json:"audio,omitempty"`              // 音频
	AudioCache       *CacheSpend           `bson:"audio_cache,omitempty"        json:"audio_cache,omitempty"`        // 音频缓存
	Video            *VideoSpend           `bson:"video,omitempty"              json:"video,omitempty"`              // 视频
	VideoGeneration  *VideoGenerationSpend `bson:"video_generation,omitempty"   json:"video_generation,omitempty"`   // 视频生成
	VideoCache       *CacheSpend           `bson:"video_cache,omitempty"        json:"video_cache,omitempty"`        // 视频缓存
	Search           *SearchSpend          `bson:"search,omitempty"             json:"search,omitempty"`             // 搜索
	Midjourney       *MidjourneySpend      `bson:"midjourney,omitempty"         json:"midjourney,omitempty"`         // Midjourney
	Once             *OnceSpend            `bson:"once,omitempty"               json:"once,omitempty"`               // 一次
	GroupId          string                `bson:"group_id,omitempty"           json:"group_id,omitempty"`           // 分组ID
	GroupName        string                `bson:"group_name,omitempty"         json:"group_name,omitempty"`         // 分组名称
	GroupDiscount    float64               `bson:"group_discount,omitempty"     json:"group_discount,omitempty"`     // 分组折扣
	TotalSpendTokens int                   `bson:"total_spend_tokens,omitempty" json:"total_spend_tokens,omitempty"` // 总花费Token数
}

type TextSpend struct {
	Pricing         *TextPricing `bson:"pricing,omitempty"          json:"pricing,omitempty"`          // 定价
	InputTokens     int          `bson:"input_tokens,omitempty"     json:"input_tokens,omitempty"`     // 输入Token数
	OutputTokens    int          `bson:"output_tokens,omitempty"    json:"output_tokens,omitempty"`    // 输出Token数
	ReasoningTokens int          `bson:"reasoning_tokens,omitempty" json:"reasoning_tokens,omitempty"` // 思考Token数
	SpendTokens     int          `bson:"spend_tokens,omitempty"     json:"spend_tokens,omitempty"`     // 花费Token数
}

type CacheSpend struct {
	Pricing     *CachePricing `bson:"pricing,omitempty"      json:"pricing,omitempty"`      // 定价
	ReadTokens  int           `bson:"read_tokens,omitempty"  json:"read_tokens,omitempty"`  // 读取/命中Token数
	WriteTokens int           `bson:"write_tokens,omitempty" json:"write_tokens,omitempty"` // 写入Token数
	SpendTokens int           `bson:"spend_tokens,omitempty" json:"spend_tokens,omitempty"` // 花费Token数
}

type ImageSpend struct {
	Pricing      *ImagePricing `bson:"pricing,omitempty"       json:"pricing,omitempty"`       // 定价
	InputTokens  int           `bson:"input_tokens,omitempty"  json:"input_tokens,omitempty"`  // 输入Token数
	OutputTokens int           `bson:"output_tokens,omitempty" json:"output_tokens,omitempty"` // 输出Token数
	SpendTokens  int           `bson:"spend_tokens,omitempty"  json:"spend_tokens,omitempty"`  // 花费Token数
}

type ImageGenerationSpend struct {
	Pricing     *ImageGenerationPricing `bson:"pricing,omitempty"      json:"pricing,omitempty"`      // 定价
	N           int                     `bson:"n,omitempty"            json:"n,omitempty"`            // 图像数
	SpendTokens int                     `bson:"spend_tokens,omitempty" json:"spend_tokens,omitempty"` // 花费Token数
}

type VisionSpend struct {
	Pricing     *VisionPricing `bson:"pricing,omitempty"      json:"pricing,omitempty"`      // 定价
	SpendTokens int            `bson:"spend_tokens,omitempty" json:"spend_tokens,omitempty"` // 花费Token数
}

type AudioSpend struct {
	Pricing      *AudioPricing `bson:"pricing,omitempty"       json:"pricing,omitempty"`       // 定价
	InputTokens  int           `bson:"input_tokens,omitempty"  json:"input_tokens,omitempty"`  // 输入Token数
	OutputTokens int           `bson:"output_tokens,omitempty" json:"output_tokens,omitempty"` // 输出Token数
	SpendTokens  int           `bson:"spend_tokens,omitempty"  json:"spend_tokens,omitempty"`  // 花费Token数
}

type VideoSpend struct {
	Pricing      *VideoPricing `bson:"pricing,omitempty"       json:"pricing,omitempty"`       // 定价
	InputTokens  int           `bson:"input_tokens,omitempty"  json:"input_tokens,omitempty"`  // 输入Token数
	OutputTokens int           `bson:"output_tokens,omitempty" json:"output_tokens,omitempty"` // 输出Token数
	SpendTokens  int           `bson:"spend_tokens,omitempty"  json:"spend_tokens,omitempty"`  // 花费Token数
}

type VideoGenerationSpend struct {
	Pricing     *VideoGenerationPricing `bson:"pricing,omitempty"      json:"pricing,omitempty"`      // 定价
	Seconds     int                     `bson:"seconds,omitempty"      json:"seconds,omitempty"`      // 秒数
	SpendTokens int                     `bson:"spend_tokens,omitempty" json:"spend_tokens,omitempty"` // 花费Token数
}

type SearchSpend struct {
	Pricing     *SearchPricing `bson:"pricing,omitempty"      json:"pricing,omitempty"`      // 定价
	SpendTokens int            `bson:"spend_tokens,omitempty" json:"spend_tokens,omitempty"` // 花费Token数
}

type MidjourneySpend struct {
	Pricing     *MidjourneyPricing `bson:"pricing,omitempty"      json:"pricing,omitempty"`      // 定价
	SpendTokens int                `bson:"spend_tokens,omitempty" json:"spend_tokens,omitempty"` // 花费Token数
}

type OnceSpend struct {
	Pricing      *OncePricing `bson:"pricing,omitempty"       json:"pricing,omitempty"`       // 定价
	SpendTokens  int          `bson:"spend_tokens,omitempty"  json:"spend_tokens,omitempty"`  // 花费Token数
	InputTokens  int          `bson:"input_tokens,omitempty"  json:"input_tokens,omitempty"`  // 输入Token数
	OutputTokens int          `bson:"output_tokens,omitempty" json:"output_tokens,omitempty"` // 输出Token数
}
