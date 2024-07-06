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
		// Main
		Main(ctx context.Context, request *ghttp.Request, retry ...int) (response sdkm.MidjourneyResponse, err error)
		// Fetch
		Fetch(ctx context.Context, request *ghttp.Request, retry ...int) (response sdkm.MidjourneyResponse, err error)
		// 保存Midjourney日志
		SaveLog(ctx context.Context, model *model.Model, key *model.Key, prompt string, response model.MidjourneyResponse)
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
