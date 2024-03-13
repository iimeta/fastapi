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
		GetModelAgent(ctx context.Context, id string) (*model.ModelAgent, error)
		// 模型代理列表
		List(ctx context.Context, ids []string) ([]*model.ModelAgent, error)
		// 根据模型代理ID获取密钥列表
		GetModelAgentKeys(ctx context.Context, id string) ([]*model.Key, error)
		// 挑选模型代理
		PickModelAgent(ctx context.Context, m *model.Model) (modelAgent *model.ModelAgent, err error)
		// 移除模型代理
		RemoveModelAgent(ctx context.Context, m *model.Model, modelAgent *model.ModelAgent)
		// 记录错误模型代理
		RecordErrorModelAgent(ctx context.Context, m *model.Model, modelAgent *model.ModelAgent)
		// 禁用错误模型代理
		DisabledModelAgent(ctx context.Context, modelAgent *model.ModelAgent)
		// 挑选模型代理密钥
		PickModelAgentKey(ctx context.Context, modelAgent *model.ModelAgent) (keyTotal int, key *model.Key, err error)
		// 移除模型代理密钥
		RemoveModelAgentKey(ctx context.Context, modelAgent *model.ModelAgent, key *model.Key)
		// 记录错误模型代理密钥
		RecordErrorModelAgentKey(ctx context.Context, modelAgent *model.ModelAgent, key *model.Key)
		// 禁用模型代理密钥
		DisabledModelAgentKey(ctx context.Context, key *model.Key)
		// 保存模型代理列表到缓存
		SaveCacheList(ctx context.Context, modelAgents []*model.ModelAgent) error
		// 获取缓存中的模型代理列表
		GetCacheList(ctx context.Context, ids ...string) ([]*model.ModelAgent, error)
		// 新增模型代理到缓存列表中
		CreateCacheModelAgent(ctx context.Context, newData *model.ModelAgent)
		// 更新缓存中的模型代理列表
		UpdateCacheModelAgent(ctx context.Context, oldData *model.ModelAgent, newData *model.ModelAgent)
		// 移除缓存中的模型代理
		RemoveCacheModelAgent(ctx context.Context, modelAgent *model.ModelAgent)
		// 保存模型代理密钥列表到缓存
		SaveCacheModelAgentKeys(ctx context.Context, id string, keys []*model.Key) error
		// 获取缓存中的模型代理密钥列表
		GetCacheModelAgentKeys(ctx context.Context, id string) ([]*model.Key, error)
		// 新增模型代理密钥到缓存列表中
		CreateCacheModelAgentKey(ctx context.Context, key *entity.Key)
		// 更新缓存中的模型代理密钥
		UpdateCacheModelAgentKey(ctx context.Context, oldData *entity.Key, newData *entity.Key)
		// 移除缓存中的模型代理密钥
		RemoveCacheModelAgentKey(ctx context.Context, key *entity.Key)
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
