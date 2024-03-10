// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/sashabaranov/go-openai"
)

type (
	IChat interface {
		// Completions
		Completions(ctx context.Context, params openai.ChatCompletionRequest, retry ...int) (response sdkm.ChatCompletionResponse, err error)
		// CompletionsStream
		CompletionsStream(ctx context.Context, params openai.ChatCompletionRequest, retry ...int) (err error)
		// 保存文生文聊天数据
		SaveChat(ctx context.Context, model *model.Model, key *model.Key, completionsReq *openai.ChatCompletionRequest, completionsRes *model.CompletionsRes)
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
