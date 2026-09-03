// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	"github.com/gogf/gf/v2/net/ghttp"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	"github.com/iimeta/fastapi/v2/internal/model"
)

type (
	IBailian interface {
		// ImageGenerations
		ImageGenerations(ctx context.Context, data []byte, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error)
		// ImageGenerationsAsync
		ImageGenerationsAsync(ctx context.Context, data []byte, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.ImageJobResponse, err error)
		// VideoCreate
		VideoCreate(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error)
		// VideoRetrieve
		VideoRetrieve(ctx context.Context, request *ghttp.Request, taskId string, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error)
	}
)

var (
	localBailian IBailian
)

func Bailian() IBailian {
	if localBailian == nil {
		panic("implement not found for interface IBailian, forgot register?")
	}
	return localBailian
}

func RegisterBailian(i IBailian) {
	localBailian = i
}
