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
	IProvider interface {
		// 根据提供商ID获取提供商信息
		GetProvider(ctx context.Context, id string) (*model.Provider, error)
		// 提供商列表
		List(ctx context.Context) ([]*model.Provider, error)
		// 根据提供商ID获取提供商信息并保存到缓存
		GetProviderAndSaveCache(ctx context.Context, id string) (*model.Provider, error)
		// 保存提供商到缓存
		SaveCache(ctx context.Context, provider *model.Provider) error
		// 保存提供商列表到缓存
		SaveCacheList(ctx context.Context, providers []*model.Provider) error
		// 获取缓存中的提供商信息
		GetCacheProvider(ctx context.Context, id string) (*model.Provider, error)
		// 获取缓存中的提供商列表
		GetCacheList(ctx context.Context, ids ...string) ([]*model.Provider, error)
		// 更新缓存中的提供商列表
		UpdateCacheProvider(ctx context.Context, newData *entity.Provider)
		// 移除缓存中的提供商列表
		RemoveCacheProvider(ctx context.Context, id string)
		// 变更订阅
		Subscribe(ctx context.Context, msg string) error
	}
)

var (
	localProvider IProvider
)

func Provider() IProvider {
	if localProvider == nil {
		panic("implement not found for interface IProvider, forgot register?")
	}
	return localProvider
}

func RegisterProvider(i IProvider) {
	localProvider = i
}
