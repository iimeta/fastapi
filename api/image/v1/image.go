package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/sashabaranov/go-openai"
)

// Generations接口请求参数
type GenerationsReq struct {
	g.Meta `path:"/generations" tags:"image" method:"post" summary:"Generations接口"`
	openai.ImageRequest
}

// Generations接口响应参数
type GenerationsRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
