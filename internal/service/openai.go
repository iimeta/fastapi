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
	IOpenAI interface {
		// Responses
		Responses(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.OpenAIResponsesRes, err error)
		// ResponsesStream
		ResponsesStream(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error)
	}
)

var (
	localOpenAI IOpenAI
)

func OpenAI() IOpenAI {
	if localOpenAI == nil {
		panic("implement not found for interface IOpenAI, forgot register?")
	}
	return localOpenAI
}

func RegisterOpenAI(i IOpenAI) {
	localOpenAI = i
}
