package model

import "github.com/iimeta/fastapi/internal/model/common"

// Subscription接口响应参数
type DashboardSubscriptionRes struct {
	Object             string  `json:"object"`
	HasPaymentMethod   bool    `json:"has_payment_method"`
	SoftLimitUSD       float64 `json:"soft_limit_usd"`
	HardLimitUSD       float64 `json:"hard_limit_usd"`
	SystemHardLimitUSD float64 `json:"system_hard_limit_usd"`
	AccessUntil        int64   `json:"access_until"`
}

// Usage接口响应参数
type DashboardUsageRes struct {
	Object     string  `json:"object"`
	TotalUsage float64 `json:"total_usage"`
}

// Models接口响应参数
type DashboardModelsRes struct {
	Object string                `json:"object"`
	Data   []DashboardModelsData `json:"data"`
}

type DashboardModelsData struct {
	Id         string       `json:"id"`
	Object     string       `json:"object"`
	OwnedBy    string       `json:"owned_by"`
	Created    int          `json:"created"`
	Root       string       `json:"root"`
	Parent     *string      `json:"parent"`
	Permission []Permission `json:"permission"`
	FastAPI    *FastAPI     `json:"fastapi,omitempty"`
}

type Permission struct {
	Id                 string  `json:"id"`
	Object             string  `json:"object"`
	Created            int     `json:"created"`
	AllowCreateEngine  bool    `json:"allow_create_engine"`
	AllowSampling      bool    `json:"allow_sampling"`
	AllowLogprobs      bool    `json:"allow_logprobs"`
	AllowSearchIndices bool    `json:"allow_search_indices"`
	AllowView          bool    `json:"allow_view"`
	AllowFineTuning    bool    `json:"allow_fine_tuning"`
	Organization       string  `json:"organization"`
	Group              *string `json:"group"`
	IsBlocking         bool    `json:"is_blocking"`
}

type FastAPI struct {
	Provider             string                      `json:"provider,omitempty"`               // 提供商名称
	Code                 string                      `json:"code,omitempty"`                   // 提供商代码
	Model                string                      `json:"model,omitempty"`                  // 模型
	Type                 int                         `json:"type,omitempty"`                   // 模型类型[1:文生文, 2:文生图, 3:图生文, 4:图生图, 5:文生语音, 6:语音生文, 7:文本向量化, 100:多模态, 101:多模态实时, 102:多模态语音, 103:多模态向量化]
	BaseUrl              string                      `json:"base_url,omitempty"`               // 模型地址
	Path                 string                      `json:"path,omitempty"`                   // 模型路径
	Pricing              common.Pricing              `json:"pricing,omitempty"`                // 定价
	TextQuota            common.TextQuota            `json:"text_quota,omitempty"`             // 文本额度
	ImageQuota           common.ImageQuota           `json:"image_quota,omitempty"`            // 图像额度
	AudioQuota           common.AudioQuota           `json:"audio_quota,omitempty"`            // 音频额度
	MultimodalQuota      common.MultimodalQuota      `json:"multimodal_quota,omitempty"`       // 多模态额度
	RealtimeQuota        common.RealtimeQuota        `json:"realtime_quota,omitempty"`         // 多模态实时额度
	MultimodalAudioQuota common.MultimodalAudioQuota `json:"multimodal_audio_quota,omitempty"` // 多模态语音额度
	MidjourneyQuotas     []common.MidjourneyQuota    `json:"midjourney_quotas,omitempty"`      // Midjourney额度
	Remark               string                      `json:"remark,omitempty"`                 // 备注
}
