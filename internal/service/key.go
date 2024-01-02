// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	"github.com/iimeta/fastapi/internal/model"
)

type (
	IKey interface {
		// 根据secretKey获取密钥信息
		GetKeyBySecretKey(ctx context.Context, secretKey string) (*model.Key, error)
		// 密钥列表
		List(ctx context.Context) ([]*model.Key, error)
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
