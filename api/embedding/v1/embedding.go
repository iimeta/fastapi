package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	sdkm "github.com/iimeta/fastapi-sdk/model"
)

// Embeddings接口请求参数
type EmbeddingsReq struct {
	g.Meta `path:"/embeddings" tags:"embedding" method:"post" summary:"embeddings接口"`
	sdkm.EmbeddingRequest
}

// Embeddings接口响应参数
type EmbeddingsRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
