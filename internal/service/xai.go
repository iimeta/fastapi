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
	IXAI interface {
		// VideoCreate
		VideoCreate(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error)
		// VideoRetrieve
		VideoRetrieve(ctx context.Context, request *ghttp.Request, requestId string, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error)
	}
)

var (
	localXAI IXAI
)

func XAI() IXAI {
	if localXAI == nil {
		panic("implement not found for interface IXAI, forgot register?")
	}
	return localXAI
}

func RegisterXAI(i IXAI) {
	localXAI = i
}
