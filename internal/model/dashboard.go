package model

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
	Id      string   `json:"id"`
	Object  string   `json:"object"`
	OwnedBy string   `json:"owned_by"`
	Created int      `json:"created"`
	FastAPI *FastAPI `json:"fastapi,omitempty"`
}

type FastAPI struct {
	Corp            string  `json:"corp,omitempty"`             // 公司名称
	Code            string  `json:"code,omitempty"`             // 公司代码
	Model           string  `json:"model,omitempty"`            // 模型
	Type            int     `json:"type,omitempty"`             // 模型类型[1:文生文, 2:文生图, 3:图生文, 4:图生图, 5:文生语音, 6:语音生文, 100:多模态]
	BaseUrl         string  `json:"base_url,omitempty"`         // 模型地址
	Path            string  `json:"path,omitempty"`             // 模型路径
	BillingMethod   int     `json:"billing_method,omitempty"`   // 计费方式[1:倍率, 2:固定额度]
	PromptRatio     float64 `json:"prompt_ratio,omitempty"`     // 提示倍率(提问倍率)
	CompletionRatio float64 `json:"completion_ratio,omitempty"` // 补全倍率(回答倍率)
	FixedQuota      int     `json:"fixed_quota,omitempty"`      // 固定额度
}
