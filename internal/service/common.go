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
	ICommon interface {
		// 核验密钥
		VerifySecretKey(ctx context.Context, secretKey string) error
		// 解析密钥
		ParseSecretKey(ctx context.Context, secretKey string) (int, int, error)
		// 记录错误次数和禁用
		RecordError(ctx context.Context, model *model.Model, key *model.Key, modelAgent *model.ModelAgent)
		// 记录使用额度
		RecordUsage(ctx context.Context, model *model.Model, usage *sdkm.Usage) error
		GetUserTotalTokens(ctx context.Context) (int, error)
		GetAppTotalTokens(ctx context.Context) (int, error)
		GetKeyTotalTokens(ctx context.Context) (int, error)
		GetUserUsageKey(ctx context.Context) string
		GetAppTotalTokensField(ctx context.Context) string
		GetKeyTotalTokensField(ctx context.Context) string
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
