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
		GetKey(ctx context.Context, secretKey string) (*model.Key, error)
		// 根据模型ID获取密钥列表
		GetModelKeys(ctx context.Context, id string) ([]*model.Key, error)
		// 密钥列表
		List(ctx context.Context, typ int) ([]*model.Key, error)
		// 挑选模型密钥
		PickModelKey(ctx context.Context, m *model.Model) (key *model.Key, err error)
		// 移除模型密钥
		RemoveModelKey(ctx context.Context, m *model.Model, key *model.Key)
		// 记录错误模型密钥
		RecordErrorModelKey(ctx context.Context, m *model.Model, key *model.Key)
		// 根据密钥更新额度
		ChangeQuota(ctx context.Context, secretKey string, quota int) error
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
