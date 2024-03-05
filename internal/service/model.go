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
	IModel interface {
		// 根据model获取模型信息
		GetModel(ctx context.Context, m string) (*model.Model, error)
		// 根据model和secretKey获取模型信息
		GetModelBySecretKey(ctx context.Context, m, secretKey string) (md *model.Model, err error)
		// 模型列表
		List(ctx context.Context, ids []string) ([]*model.Model, error)
		// 保存模型列表到缓存
		SaveCacheList(ctx context.Context, models []*model.Model) error
		// 获取缓存中的模型列表
		GetCacheList(ctx context.Context, ids ...string) ([]*model.Model, error)
		// 更新缓存中的模型列表
		UpdateCacheModel(ctx context.Context, oldData *entity.Model, newData *entity.Model)
		// 移除缓存中的模型列表
		RemoveCacheModel(ctx context.Context, id string)
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
