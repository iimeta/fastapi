// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	smodel "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/model"
)

type (
	IEmbedding interface {
		// Embeddings
		Embeddings(ctx context.Context, data []byte, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.EmbeddingResponse, err error)
	}
)

var (
	localEmbedding IEmbedding
)

func Embedding() IEmbedding {
	if localEmbedding == nil {
		panic("implement not found for interface IEmbedding, forgot register?")
	}
	return localEmbedding
}

func RegisterEmbedding(i IEmbedding) {
	localEmbedding = i
}
