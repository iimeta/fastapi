// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/model"
)

type (
	IMidjourney interface {
		Imagine(ctx context.Context, params sdkm.MidjourneyProxyRequest, retry ...int) (response sdkm.MidjourneyProxyResponse, err error)
		Change(ctx context.Context, params sdkm.MidjourneyProxyRequest, retry ...int) (sdkm.MidjourneyProxyResponse, error)
		Describe(ctx context.Context, params sdkm.MidjourneyProxyRequest, retry ...int) (sdkm.MidjourneyProxyResponse, error)
		Blend(ctx context.Context, params sdkm.MidjourneyProxyRequest, retry ...int) (sdkm.MidjourneyProxyResponse, error)
		Fetch(ctx context.Context, params sdkm.MidjourneyProxyRequest, retry ...int) (sdkm.MidjourneyProxyFetchResponse, error)
		// 保存Midjourney数据
		SaveChat(ctx context.Context, model *model.Model, key *model.Key, request sdkm.MidjourneyProxyRequest, response model.MidjourneyProxyResponse)
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
