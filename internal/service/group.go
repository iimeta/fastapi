// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	"github.com/iimeta/fastapi/v2/internal/model"
	"github.com/iimeta/fastapi/v2/internal/model/entity"
)

type (
	IGroup interface {
		// 根据分组ID获取分组信息
		GetById(ctx context.Context, id string) (*model.Group, error)
		// 分组列表
		List(ctx context.Context) ([]*model.Group, error)
		// 根据分组ID获取分组信息并保存到缓存
		GetAndSaveCache(ctx context.Context, id string) (*model.Group, error)
		// 保存分组到缓存
		SaveCache(ctx context.Context, group *model.Group) error
		// 保存分组列表到缓存
		SaveCacheList(ctx context.Context, groups []*model.Group) error
		// 获取缓存中的分组信息
		GetCache(ctx context.Context, id string) (*model.Group, error)
		// 获取缓存中的分组列表
		GetCacheList(ctx context.Context, ids ...string) ([]*model.Group, error)
		// 更新缓存中的分组列表
		UpdateCache(ctx context.Context, newData *entity.Group)
		// 移除缓存中的分组列表
		RemoveCache(ctx context.Context, id string)
		// 根据分组Ids获取模型Ids
		GetModelIds(ctx context.Context, ids ...string) ([]string, error)
		// 根据分组Ids获取默认分组
		GetDefault(ctx context.Context, ids ...string) (*model.Group, error)
		// 根据model挑选分组和模型
		PickGroupAndModel(ctx context.Context, m string, ids ...string) (reqModel *model.Model, group *model.Group, err error)
		// 分组花费额度
		SpendQuota(ctx context.Context, group string, spendQuota int, currentQuota int) error
		// 分组已用额度
		UsedQuota(ctx context.Context, group string, quota int) error
		// 保存分组额度到缓存
		SaveCacheQuota(ctx context.Context, group string, quota int) error
		// 获取缓存中的分组额度
		GetCacheQuota(ctx context.Context, group string) int
		// 变更订阅
		Subscribe(ctx context.Context, msg string) error
	}
)

var (
	localGroup IGroup
)

func Group() IGroup {
	if localGroup == nil {
		panic("implement not found for interface IGroup, forgot register?")
	}
	return localGroup
}

func RegisterGroup(i IGroup) {
	localGroup = i
}
