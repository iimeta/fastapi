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
	IReseller interface {
		// 根据用户ID获取代理商信息
		GetByUserId(ctx context.Context, userId int) (*model.Reseller, error)
		// 代理商列表
		List(ctx context.Context) ([]*model.Reseller, error)
		// 代理商花费额度
		SpendQuota(ctx context.Context, userId int, spendQuota int, currentQuota int) error
		// 保存代理商信息到缓存
		SaveCache(ctx context.Context, reseller *model.Reseller) error
		// 获取缓存中的代理商信息
		GetCache(ctx context.Context, userId int) (*model.Reseller, error)
		// 更新缓存中的代理商信息
		UpdateCache(ctx context.Context, reseller *entity.Reseller)
		// 移除缓存中的代理商信息
		RemoveCache(ctx context.Context, userId int)
		// 保存代理商额度到缓存
		SaveCacheQuota(ctx context.Context, userId int, quota int) error
		// 获取缓存中的代理商额度
		GetCacheQuota(ctx context.Context, userId int) int
		// 变更订阅
		Subscribe(ctx context.Context, msg string) error
	}
)

var (
	localReseller IReseller
)

func Reseller() IReseller {
	if localReseller == nil {
		panic("implement not found for interface IReseller, forgot register?")
	}
	return localReseller
}

func RegisterReseller(i IReseller) {
	localReseller = i
}
