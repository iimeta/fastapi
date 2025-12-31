// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	"github.com/iimeta/fastapi/v2/internal/model"
)

type (
	IModeration interface {
		// Moderations
		Moderations(ctx context.Context, params smodel.ModerationRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.ModerationResponse, err error)
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
