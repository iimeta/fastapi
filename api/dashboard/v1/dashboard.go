package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/iimeta/fastapi/internal/model"
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
