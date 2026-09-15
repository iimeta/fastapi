// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/iimeta/fastapi/v2/internal/model"
)

type (
	IMiniMax interface {
		// VideoCreate
		VideoCreate(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error)
		// VideoRetrieve
		VideoRetrieve(ctx context.Context, request *ghttp.Request, taskId string, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error)
	}
)

var (
	localMiniMax IMiniMax
)

func MiniMax() IMiniMax {
	if localMiniMax == nil {
		panic("implement not found for interface IMiniMax, forgot register?")
	}
	return localMiniMax
}

func RegisterMiniMax(i IMiniMax) {
	localMiniMax = i
}
