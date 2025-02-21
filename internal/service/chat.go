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
		Completions(ctx context.Context, params sdkm.ChatCompletionRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.ChatCompletionResponse, err error)
		// CompletionsStream
		CompletionsStream(ctx context.Context, params sdkm.ChatCompletionRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error)
		// 保存日志
		SaveLog(ctx context.Context, reqModel *model.Model, realModel *model.Model, modelAgent *model.ModelAgent, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, key *model.Key, completionsReq *sdkm.ChatCompletionRequest, completionsRes *model.CompletionsRes, retryInfo *mcommon.Retry, isSmartMatch bool, retry ...int)
		// SmartCompletions
		SmartCompletions(ctx context.Context, params sdkm.ChatCompletionRequest, reqModel *model.Model, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.ChatCompletionResponse, err error)
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
