package v1

import (
	"github.com/gogf/gf/v2/frame/g"
)

// Completions接口请求参数
type CompletionsReq struct {
	g.Meta `path:"/models/{model}" tags:"google" method:"post" summary:"Completions接口"`
	Alt    string `json:"alt"`
	Key    string `json:"key"`
}

// Completions接口响应参数
type CompletionsRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
