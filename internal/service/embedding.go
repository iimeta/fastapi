// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
)

type (
	IEmbedding interface {
		// Embeddings
		Embeddings(ctx context.Context, params sdkm.EmbeddingRequest, fallbackModel *model.Model, retry ...int) (response sdkm.EmbeddingResponse, err error)
		// 保存日志
		SaveLog(ctx context.Context, reqModel, realModel, fallbackModel *model.Model, key *model.Key, completionsReq *sdkm.EmbeddingRequest, completionsRes *model.CompletionsRes, retryInfo *mcommon.Retry, isSmartMatch ...bool)
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
