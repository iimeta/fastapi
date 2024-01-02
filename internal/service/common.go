// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
)

type (
	ICommon interface {
		VerifySecretKey(ctx context.Context, secretKey string) (bool, error)
		GetUidUsageKey(ctx context.Context) string
		RecordUsage(ctx context.Context, totalTokens int) error
		GetUsageCount(ctx context.Context) (int, error)
		GetUsedTokens(ctx context.Context) (int, error)
		GetTotalTokens(ctx context.Context) (int, error)
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
