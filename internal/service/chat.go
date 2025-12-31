// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	"github.com/iimeta/fastapi/v2/internal/model"
)

type (
	IChat interface {
		// Completions
		Completions(ctx context.Context, params smodel.ChatCompletionRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.ChatCompletionResponse, err error)
		// CompletionsStream
		CompletionsStream(ctx context.Context, params smodel.ChatCompletionRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error)
		// SmartCompletions
		SmartCompletions(ctx context.Context, params smodel.ChatCompletionRequest, reqModel *model.Model, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.ChatCompletionResponse, err error)
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
