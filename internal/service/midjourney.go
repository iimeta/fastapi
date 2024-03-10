// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	"github.com/iimeta/fastapi-sdk/model"
)

type (
	IMidjourney interface {
		Imagine(ctx context.Context, midjourneyProxy *model.MidjourneyProxy, midjourneyProxyImagineReq *model.MidjourneyProxyImagineReq) (*model.MidjourneyProxyImagineRes, error)
		Change(ctx context.Context, midjourneyProxy *model.MidjourneyProxy, midjourneyProxyChangeReq *model.MidjourneyProxyChangeReq) (*model.MidjourneyProxyChangeRes, error)
		Describe(ctx context.Context, midjourneyProxy *model.MidjourneyProxy, midjourneyProxyDescribeReq *model.MidjourneyProxyDescribeReq) (*model.MidjourneyProxyDescribeRes, error)
		Blend(ctx context.Context, midjourneyProxy *model.MidjourneyProxy, midjourneyProxyBlendReq *model.MidjourneyProxyBlendReq) (*model.MidjourneyProxyBlendRes, error)
		Fetch(ctx context.Context, midjourneyProxy *model.MidjourneyProxy, taskId string) (*model.MidjourneyProxyFetchRes, error)
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
