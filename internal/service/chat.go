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
	IChat interface {
		// Completions
		Completions(ctx context.Context, params sdkm.ChatCompletionRequest, fallbackModel *model.Model, retry ...int) (response sdkm.ChatCompletionResponse, err error)
		// CompletionsStream
		CompletionsStream(ctx context.Context, params sdkm.ChatCompletionRequest, fallbackModel *model.Model, retry ...int) (err error)
		// 保存文生文聊天数据
		SaveChat(ctx context.Context, reqModel, realModel, fallbackModel *model.Model, key *model.Key, completionsReq *sdkm.ChatCompletionRequest, completionsRes *model.CompletionsRes, retryInfo *mcommon.Retry, isSmartMatch ...bool)
		// SmartCompletions
		SmartCompletions(ctx context.Context, params sdkm.ChatCompletionRequest, reqModel *model.Model, fallbackModel *model.Model, retry ...int) (response sdkm.ChatCompletionResponse, err error)
	}
)

var (
	localChat IChat
)

func Chat() IChat {
	if localChat == nil {
		panic("implement not found for interface IChat, forgot register?")
	}
	return localChat
}

func RegisterChat(i IChat) {
	localChat = i
}
