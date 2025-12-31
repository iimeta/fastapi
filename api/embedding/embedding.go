// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package embedding

import (
	"context"

	"github.com/iimeta/fastapi/v2/api/embedding/v1"
)

type IEmbeddingV1 interface {
	Embeddings(ctx context.Context, req *v1.EmbeddingsReq) (res *v1.EmbeddingsRes, err error)
}
