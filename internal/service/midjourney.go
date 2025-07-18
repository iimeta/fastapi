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
	IMidjourney interface {
		// 任务提交
		Submit(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.MidjourneyResponse, err error)
		// 任务查询
		Task(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.MidjourneyResponse, err error)
		// 保存日志
		SaveLog(ctx context.Context, midjourneyLog model.MidjourneyLog, retry ...int)
	}
)

var (
	localMidjourney IMidjourney
)

func Midjourney() IMidjourney {
	if localMidjourney == nil {
		panic("implement not found for interface IMidjourney, forgot register?")
	}
	return localMidjourney
}

func RegisterMidjourney(i IMidjourney) {
	localMidjourney = i
}
