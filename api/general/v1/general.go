package v1

import (
	"github.com/gogf/gf/v2/frame/g"
)

// General接口请求参数
type GeneralReq struct {
	g.Meta `path:"/*" tags:"general" method:"post" summary:"General接口"`
	Stream bool `json:"stream"`
}

// General接口响应参数
type GeneralRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
