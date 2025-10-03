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
	IModelAgent interface {
		// 根据模型代理ID获取模型代理信息
		GetById(ctx context.Context, id string) (*model.ModelAgent, error)
		// 模型代理列表
		List(ctx context.Context, ids []string) ([]*model.ModelAgent, error)
		// 全部模型代理列表
		ListAll(ctx context.Context) ([]*model.ModelAgent, error)
		// 根据模型代理ID获取密钥列表
		GetKeys(ctx context.Context, id string) ([]*model.Key, error)
		// 获取模型代理与密钥列表
		GetModelAgentsAndKeys(ctx context.Context) ([]*model.ModelAgent, map[string][]*model.Key, error)
		// 挑选模型代理
		Pick(ctx context.Context, m *model.Model) (int, *model.ModelAgent, error)
		// 根据模型挑选分组模型代理
		PickGroup(ctx context.Context, m *model.Model, group *model.Group) (int, *model.ModelAgent, error)
		// 移除模型代理
		Remove(ctx context.Context, m *model.Model, modelAgent *model.ModelAgent)
		// 记录错误模型代理
		RecordError(ctx context.Context, m *model.Model, modelAgent *model.ModelAgent)
		// 禁用模型代理
		Disabled(ctx context.Context, modelAgent *model.ModelAgent, disabledReason string)
		// 挑选模型代理密钥
		PickKey(ctx context.Context, modelAgent *model.ModelAgent) (int, *model.Key, error)
		// 移除模型代理密钥
		RemoveKey(ctx context.Context, modelAgent *model.ModelAgent, key *model.Key)
		// 记录错误模型代理密钥
		RecordErrorKey(ctx context.Context, modelAgent *model.ModelAgent, key *model.Key)
		// 禁用模型代理密钥
		DisabledKey(ctx context.Context, key *model.Key, disabledReason string)
		// 保存模型代理列表到缓存
		SaveCacheList(ctx context.Context, modelAgents []*model.ModelAgent) error
		// 获取缓存中的模型代理列表
		GetCacheList(ctx context.Context, ids ...string) ([]*model.ModelAgent, error)
		// 添加模型代理到缓存列表中
		AddCache(ctx context.Context, newData *model.ModelAgent)
		// 更新缓存中的模型代理列表
		UpdateCache(ctx context.Context, oldData *entity.ModelAgent, newData *model.ModelAgent)
		// 移除缓存中的模型代理
		RemoveCache(ctx context.Context, modelAgent *model.ModelAgent)
		// 保存模型代理密钥列表到缓存
		SaveCacheKeys(ctx context.Context, id string, keys []*model.Key) error
		// 获取缓存中的模型代理密钥列表
		GetCacheKeys(ctx context.Context, id string) ([]*model.Key, error)
		// 新增模型代理密钥到缓存列表中
		CreateCacheKey(ctx context.Context, key *entity.Key)
		// 更新缓存中的模型代理密钥
		UpdateCacheKey(ctx context.Context, oldData *entity.Key, newData *entity.Key)
		// 移除缓存中的模型代理密钥
		RemoveCacheKey(ctx context.Context, key *entity.Key)
		// 获取缓存中的模型代理信息
		GetCache(ctx context.Context, id string) (*model.ModelAgent, error)
		// 根据模型代理ID获取模型代理信息并保存到缓存
		GetAndSaveCache(ctx context.Context, id string) (*model.ModelAgent, error)
		// 保存模型代理到缓存
		SaveCache(ctx context.Context, modelAgent *model.ModelAgent) error
		// 获取后备模型代理
		GetFallback(ctx context.Context, model *model.Model) (fallbackModelAgent *model.ModelAgent, err error)
		// 保存分组模型代理列表到缓存
		SaveGroupCache(ctx context.Context, group *model.Group) error
		// 变更订阅
		Subscribe(ctx context.Context, msg string) error
	}
)

var (
	localModelAgent IModelAgent
)

func ModelAgent() IModelAgent {
	if localModelAgent == nil {
		panic("implement not found for interface IModelAgent, forgot register?")
	}
	return localModelAgent
}

func RegisterModelAgent(i IModelAgent) {
	localModelAgent = i
}
