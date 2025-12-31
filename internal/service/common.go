// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	"github.com/iimeta/fastapi/v2/internal/model"
)

type (
	ICommon interface {
		// 解析密钥
		ParseSecretKey(ctx context.Context, secretKey string) (int, int, error)
		// 记录错误次数和禁用
		RecordError(ctx context.Context, model *model.Model, key *model.Key, modelAgent *model.ModelAgent)
	}
)

var (
	localCommon ICommon
)

func Common() ICommon {
	if localCommon == nil {
		panic("implement not found for interface ICommon, forgot register?")
	}
	return localCommon
}

func RegisterCommon(i ICommon) {
	localCommon = i
}
