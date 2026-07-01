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
	IKey interface {
		// 密钥列表
		List(ctx context.Context) ([]*model.Key, error)
		// 密钥已用额度
		UsedQuota(ctx context.Context, key string, quota int) error
		// 变更订阅
		Subscribe(ctx context.Context, msg string) error
	}
)

var (
	localKey IKey
)

func Key() IKey {
	if localKey == nil {
		panic("implement not found for interface IKey, forgot register?")
	}
	return localKey
}

func RegisterKey(i IKey) {
	localKey = i
}
