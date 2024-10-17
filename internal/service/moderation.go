// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
)

type (
	IModeration interface {
		// Moderations
		Moderations(ctx context.Context, params sdkm.ModerationRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.ModerationResponse, err error)
		// 保存日志
		SaveLog(ctx context.Context, reqModel *model.Model, realModel *model.Model, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, key *model.Key, completionsReq *sdkm.ModerationRequest, completionsRes *model.CompletionsRes, retryInfo *mcommon.Retry, retry ...int)
	}
)

var (
	localModeration IModeration
)

func Moderation() IModeration {
	if localModeration == nil {
		panic("implement not found for interface IModeration, forgot register?")
	}
	return localModeration
}

func RegisterModeration(i IModeration) {
	localModeration = i
}
