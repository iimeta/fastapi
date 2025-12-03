package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	smodel "github.com/iimeta/fastapi-sdk/model"
)

// Embeddings接口请求参数
type EmbeddingsReq struct {
	g.Meta `path:"/embeddings" tags:"embedding" method:"post" summary:"Embeddings接口"`
	smodel.EmbeddingRequest
}

// Embeddings接口响应参数
type EmbeddingsRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
