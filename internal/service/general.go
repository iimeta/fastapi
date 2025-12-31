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
	IGeneral interface {
		// General
		General(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.ChatCompletionResponse, err error)
		// GeneralStream
		GeneralStream(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error)
	}
)

var (
	localGeneral IGeneral
)

func General() IGeneral {
	if localGeneral == nil {
		panic("implement not found for interface IGeneral, forgot register?")
	}
	return localGeneral
}

func RegisterGeneral(i IGeneral) {
	localGeneral = i
}
