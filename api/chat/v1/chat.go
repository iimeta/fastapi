package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	sdkm "github.com/iimeta/fastapi-sdk/model"
)

// ChatCompletions接口请求参数
type CompletionsReq struct {
	g.Meta `path:"/completions" tags:"chat" method:"post" summary:"ChatCompletions接口"`
	sdkm.ChatCompletionRequest
	IsToResponses bool `json:"is_to_responses"`
}

// ChatCompletions接口响应参数
type CompletionsRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// ChatCompletionsToResponses接口请求参数
type CompletionsResponsesReq struct {
	g.Meta `path:"/completions/responses" tags:"chat" method:"post" summary:"ChatCompletionsToResponses接口"`
	sdkm.ChatCompletionRequest
}

// ChatCompletionsToResponses接口响应参数
type CompletionsResponsesRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
