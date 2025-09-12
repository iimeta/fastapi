// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/entity"
)

type (
	IKey interface {
		// 根据模型ID获取密钥列表
		GetByModelId(ctx context.Context, modelId string) ([]*model.Key, error)
		// 密钥列表
		List(ctx context.Context) ([]*model.Key, error)
		// 挑选模型密钥
		Pick(ctx context.Context, m *model.Model) (int, *model.Key, error)
		// 移除模型密钥
		Remove(ctx context.Context, m *model.Model, key *model.Key)
		// 记录错误模型密钥
		RecordError(ctx context.Context, m *model.Model, key *model.Key)
		// 禁用模型密钥
		Disabled(ctx context.Context, key *model.Key, disabledReason string)
		// 保存模型密钥列表到缓存
		SaveCache(ctx context.Context, id string, keys []*model.Key) error
		// 获取缓存中的模型密钥列表
		GetCache(ctx context.Context, id string) ([]*model.Key, error)
		// 添加模型密钥到缓存列表中
		AddCache(ctx context.Context, key *entity.Key)
		// 更新缓存中的模型密钥
		UpdateCache(ctx context.Context, oldData *entity.Key, newData *entity.Key)
		// 移除缓存中的模型密钥
		RemoveCache(ctx context.Context, key *entity.Key)
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
