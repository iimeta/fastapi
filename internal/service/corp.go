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
	ICorp interface {
		// 根据公司ID获取公司信息
		GetCorp(ctx context.Context, id string) (*model.Corp, error)
		// 公司列表
		List(ctx context.Context) ([]*model.Corp, error)
		// 根据公司ID获取公司信息并保存到缓存
		GetCorpAndSaveCache(ctx context.Context, id string) (*model.Corp, error)
		// 保存公司到缓存
		SaveCache(ctx context.Context, m *model.Corp) error
		// 保存公司列表到缓存
		SaveCacheList(ctx context.Context, corps []*model.Corp) error
		// 获取缓存中的公司信息
		GetCacheCorp(ctx context.Context, id string) (*model.Corp, error)
		// 获取缓存中的公司列表
		GetCacheList(ctx context.Context, ids ...string) ([]*model.Corp, error)
		// 更新缓存中的公司列表
		UpdateCacheCorp(ctx context.Context, newData *entity.Corp)
		// 移除缓存中的公司列表
		RemoveCacheCorp(ctx context.Context, id string)
		// 变更订阅
		Subscribe(ctx context.Context, msg string) error
	}
)

var (
	localCorp ICorp
)

func Corp() ICorp {
	if localCorp == nil {
		panic("implement not found for interface ICorp, forgot register?")
	}
	return localCorp
}

func RegisterCorp(i ICorp) {
	localCorp = i
}
