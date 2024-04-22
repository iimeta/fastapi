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
