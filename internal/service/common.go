// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	"github.com/iimeta/fastapi/internal/model"
	"github.com/sashabaranov/go-openai"
)

type (
	ICommon interface {
		VerifySecretKey(ctx context.Context, secretKey string) (bool, error)
		RecordUsage(ctx context.Context, model *model.Model, usage openai.Usage) error
		GetUserUsageKey(ctx context.Context) string
		GetAppUsageCountField(ctx context.Context) string
		GetAppUsedTokensField(ctx context.Context) string
		GetAppTotalTokensField(ctx context.Context) string
		GetKeyUsageCountField(ctx context.Context) string
		GetKeyUsedTokensField(ctx context.Context) string
		GetKeyTotalTokensField(ctx context.Context) string
		GetUserUsageCount(ctx context.Context) (int, error)
		GetUserUsedTokens(ctx context.Context) (int, error)
		GetUserTotalTokens(ctx context.Context) (int, error)
		GetAppUsageCount(ctx context.Context) (int, error)
		GetAppUsedTokens(ctx context.Context) (int, error)
		GetAppTotalTokens(ctx context.Context) (int, error)
		GetKeyUsageCount(ctx context.Context) (int, error)
		GetKeyUsedTokens(ctx context.Context) (int, error)
		GetKeyTotalTokens(ctx context.Context) (int, error)
		ParseSecretKey(ctx context.Context, secretKey string) (int, int, error)
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
