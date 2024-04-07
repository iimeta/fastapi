package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	sdkm "github.com/iimeta/fastapi-sdk/model"
)

// Completions接口请求参数
type CompletionsReq struct {
	g.Meta `path:"/completions" tags:"chat" method:"post" summary:"Completions接口"`
	sdkm.ChatCompletionRequest
}

// Completions接口响应参数
type CompletionsRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
