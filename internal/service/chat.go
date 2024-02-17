// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/model"
)

type (
	IChat interface {
		Completions(ctx context.Context, params model.CompletionsReq, retry ...int) (response sdkm.ChatCompletionResponse, err error)
		CompletionsStream(ctx context.Context, params model.CompletionsReq, retry ...int) (err error)
		SaveChat(ctx context.Context, model *model.Model, key *model.Key, completionsReq model.CompletionsReq, completionsRes model.CompletionsRes)
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
