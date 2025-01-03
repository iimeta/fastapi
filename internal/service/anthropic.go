// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	"github.com/gogf/gf/v2/net/ghttp"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/model"
)

type (
	IAnthropic interface {
		// Completions
		Completions(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.ChatCompletionResponse, err error)
		// CompletionsStream
		CompletionsStream(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error)
	}
)

var (
	localAnthropic IAnthropic
)

func Anthropic() IAnthropic {
	if localAnthropic == nil {
		panic("implement not found for interface IAnthropic, forgot register?")
	}
	return localAnthropic
}

func RegisterAnthropic(i IAnthropic) {
	localAnthropic = i
}
