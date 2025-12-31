package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/iimeta/fastapi/v2/internal/model"
)

// Subscription接口请求参数
type SubscriptionReq struct {
	g.Meta `path:"/billing/subscription" tags:"dashboard" method:"get,post" summary:"Subscription接口"`
}

// Subscription接口响应参数
type SubscriptionRes struct {
	g.Meta `mime:"application/json" example:"json"`
	*model.DashboardSubscriptionRes
}

// Usage接口请求参数
type UsageReq struct {
	g.Meta `path:"/billing/usage" tags:"dashboard" method:"get,post" summary:"Usage接口"`
}

// Usage接口响应参数
type UsageRes struct {
	g.Meta `mime:"application/json" example:"json"`
	*model.DashboardUsageRes
}

// Models接口请求参数
type ModelsReq struct {
	g.Meta    `path:"/models" tags:"dashboard" method:"get,post" summary:"Models接口"`
	IsFastAPI bool `json:"is_fastapi"`
}

// Models接口响应参数
type ModelsRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
