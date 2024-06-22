package v1

import (
	"github.com/gogf/gf/v2/frame/g"
)

// 健康接口请求参数
type HealthReq struct {
	g.Meta `path:"/health" tags:"health" method:"all" summary:"健康接口"`
}

// 健康接口响应参数
type HealthRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
