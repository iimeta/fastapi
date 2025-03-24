package v1

import (
	"github.com/gogf/gf/v2/frame/g"
)

// Responses接口请求参数
type ResponsesReq struct {
	g.Meta `path:"/responses" tags:"openai" method:"post" summary:"Responses接口"`
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
}

// Responses接口响应参数
type ResponsesRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
