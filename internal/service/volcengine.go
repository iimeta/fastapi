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
	IVolcEngine interface {
		// VideoCreate
		VideoCreate(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error)
		// VideoList
		VideoList(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error)
		// VideoRetrieve
		VideoRetrieve(ctx context.Context, request *ghttp.Request, taskId string, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error)
		// VideoDelete
		VideoDelete(ctx context.Context, request *ghttp.Request, taskId string, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error)
	}
)

var (
	localVolcEngine IVolcEngine
)

func VolcEngine() IVolcEngine {
	if localVolcEngine == nil {
		panic("implement not found for interface IVolcEngine, forgot register?")
	}
	return localVolcEngine
}

func RegisterVolcEngine(i IVolcEngine) {
	localVolcEngine = i
}
