package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/iimeta/fastapi/internal/model"
)

// 使用量接口请求参数
type UsageReq struct {
	g.Meta `path:"/usage" tags:"token" method:"get" summary:"使用量接口"`
}

// 使用量接口响应参数
type UsageRes struct {
	g.Meta `mime:"application/json" example:"json"`
	*model.UsageRes
}
