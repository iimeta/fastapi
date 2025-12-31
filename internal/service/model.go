// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	"github.com/iimeta/fastapi/v2/internal/model"
	"github.com/iimeta/fastapi/v2/internal/model/entity"
)

type (
	IModel interface {
		// 根据model获取模型信息
		GetModel(ctx context.Context, m string) (*model.Model, error)
		// 根据模型ID获取模型信息
		GetModelById(ctx context.Context, id string) (*model.Model, error)
		// 根据model和group获取模型信息
		GetModelByGroup(ctx context.Context, m string, group *model.Group) (*model.Model, error)
		// 模型列表
		List(ctx context.Context, ids []string) ([]*model.Model, error)
		// 全部模型列表
		ListAll(ctx context.Context) ([]*model.Model, error)
		// 获取模型与密钥列表
		GetModelsAndKeys(ctx context.Context) ([]*model.Model, map[string][]*model.Key, error)
		// 根据模型ID获取模型信息并保存到缓存
		GetModelAndSaveCache(ctx context.Context, id string) (*model.Model, error)
		// 获取模型列表并保存到缓存
		GetModelListAndSaveCacheList(ctx context.Context, ids []string) ([]*model.Model, error)
		// 保存模型到缓存
		SaveCache(ctx context.Context, m *model.Model) error
		// 保存模型列表到缓存
		SaveCacheList(ctx context.Context, models []*model.Model) error
		// 获取缓存中的模型列表
		GetCacheList(ctx context.Context, ids ...string) ([]*model.Model, error)
		// 获取缓存中的模型信息
		GetCacheModel(ctx context.Context, id string) (*model.Model, error)
		// 更新缓存中的模型列表
		UpdateCacheModel(ctx context.Context, oldData *entity.Model, newData *entity.Model)
		// 移除缓存中的模型列表
		RemoveCacheModel(ctx context.Context, id string)
		// 获取目标模型
		GetTargetModel(ctx context.Context, model *model.Model, messages []smodel.ChatCompletionMessage) (targetModel *model.Model, err error)
		// 获取分组目标模型
		GetGroupTargetModel(ctx context.Context, group *model.Group, model *model.Model, messages []smodel.ChatCompletionMessage) (targetModel *model.Model, err error)
		// 获取后备模型
		GetFallbackModel(ctx context.Context, model *model.Model) (fallbackModel *model.Model, err error)
		// 变更订阅
		Subscribe(ctx context.Context, msg string) error
	}
)

var (
	localModel IModel
)

func Model() IModel {
	if localModel == nil {
		panic("implement not found for interface IModel, forgot register?")
	}
	return localModel
}

func RegisterModel(i IModel) {
	localModel = i
}
