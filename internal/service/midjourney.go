// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	sdkm "github.com/iimeta/fastapi-sdk/model"
)

type (
	IMidjourney interface {
		Imagine(ctx context.Context, params sdkm.MidjourneyProxyImagineReq, retry ...int) (response sdkm.MidjourneyProxyImagineRes, err error)
		Change(ctx context.Context, params sdkm.MidjourneyProxyChangeReq) (sdkm.MidjourneyProxyChangeRes, error)
		Describe(ctx context.Context, params sdkm.MidjourneyProxyDescribeReq) (sdkm.MidjourneyProxyDescribeRes, error)
		Blend(ctx context.Context, params sdkm.MidjourneyProxyBlendReq) (sdkm.MidjourneyProxyBlendRes, error)
		Fetch(ctx context.Context, taskId string) (sdkm.MidjourneyProxyFetchRes, error)
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
