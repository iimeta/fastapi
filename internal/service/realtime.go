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
	mcommon "github.com/iimeta/fastapi/internal/model/common"
)

type (
	IRealtime interface {
		// Realtime
		Realtime(ctx context.Context, r *ghttp.Request, params model.RealtimeRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error)
		// 保存日志
		SaveLog(ctx context.Context, group *model.Group, reqModel *model.Model, realModel *model.Model, modelAgent *model.ModelAgent, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, key *model.Key, completionsReq *sdkm.ChatCompletionRequest, completionsRes *model.CompletionsRes, retryInfo *mcommon.Retry, isSmartMatch bool, retry ...int)
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
