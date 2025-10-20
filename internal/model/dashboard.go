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
	Provider string         `json:"provider,omitempty"` // 提供商名称
	Code     string         `json:"code,omitempty"`     // 提供商代码
	Model    string         `json:"model,omitempty"`    // 模型
	Type     int            `json:"type,omitempty"`     // 模型类型
	BaseUrl  string         `json:"base_url,omitempty"` // 模型地址
	Path     string         `json:"path,omitempty"`     // 模型路径
	Pricing  common.Pricing `json:"pricing,omitempty"`  // 定价
	Remark   string         `json:"remark,omitempty"`   // 备注
}
