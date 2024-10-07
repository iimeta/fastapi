// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/iimeta/fastapi/internal/model"
)

type (
	IRealtime interface {
		// Realtime
		Realtime(ctx context.Context, r *ghttp.Request, params model.RealtimeRequest, fallbackModel *model.Model, retry ...int) (err error)
	}
)

var (
	localRealtime IRealtime
)

func Realtime() IRealtime {
	if localRealtime == nil {
		panic("implement not found for interface IRealtime, forgot register?")
	}
	return localRealtime
}

func RegisterRealtime(i IRealtime) {
	localRealtime = i
}
