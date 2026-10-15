package v1

import (
	"github.com/gogf/gf/v2/frame/g"
)

// Completions接口请求参数
type CompletionsReq struct {
	g.Meta `path:"/models/{model}:{action}" tags:"google" method:"post" summary:"Completions接口"`
	Model  string `json:"model"`
	Action string `json:"action"`
	Alt    string `json:"alt"`
}

// Completions接口响应参数
type CompletionsRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// Interactions接口请求参数
type InteractionsReq struct {
	g.Meta `path:"/interactions" tags:"google" method:"post" summary:"Interactions接口"`
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
	Alt    string `json:"alt"`
}

// Interactions接口响应参数
type InteractionsRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// InteractionsRetrieve接口请求参数
type InteractionsRetrieveReq struct {
	g.Meta `path:"/interactions/{interaction_id}" tags:"google" method:"get" summary:"Interactions查询接口"`
	Id     string `json:"interaction_id"`
}

// InteractionsRetrieve接口响应参数
type InteractionsRetrieveRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
