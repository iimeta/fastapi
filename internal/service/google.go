// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	"github.com/gogf/gf/v2/net/ghttp"
	smodel "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/model"
)

type (
	IGoogle interface {
		// Completions
		Completions(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.ChatCompletionResponse, err error)
		// CompletionsStream
		CompletionsStream(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error)
	}
)

var (
	localGoogle IGoogle
)

func Google() IGoogle {
	if localGoogle == nil {
		panic("implement not found for interface IGoogle, forgot register?")
	}
	return localGoogle
}

func RegisterGoogle(i IGoogle) {
	localGoogle = i
}
