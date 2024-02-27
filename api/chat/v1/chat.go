package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/sashabaranov/go-openai"
)

// Completions接口请求参数
type CompletionsReq struct {
	g.Meta `path:"/completions" tags:"chat" method:"post" summary:"Completions接口"`
	openai.ChatCompletionRequest
}

// Completions接口响应参数
type CompletionsRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
