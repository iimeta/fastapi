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
	IApp interface {
		// 根据应用ID获取应用信息
		GetApp(ctx context.Context, appId int) (*model.App, error)
		// 应用列表
		List(ctx context.Context) ([]*model.App, error)
		// 应用花费额度
		SpendQuota(ctx context.Context, appId int, spendQuota int, currentQuota int) error
		// 应用已用额度
		UsedQuota(ctx context.Context, appId int, quota int) error
		// 保存应用额度到缓存
		SaveCacheAppQuota(ctx context.Context, appId int, quota int) error
		// 获取缓存中的应用额度
		GetCacheAppQuota(ctx context.Context, appId int) int
		// 保存应用信息到缓存
		SaveCacheApp(ctx context.Context, app *model.App) error
		// 获取缓存中的应用信息
		GetCacheApp(ctx context.Context, appId int) (*model.App, error)
		// 更新缓存中的应用信息
		UpdateCacheApp(ctx context.Context, app *entity.App)
		// 移除缓存中的应用信息
		RemoveCacheApp(ctx context.Context, appId int)
		// 变更订阅
		Subscribe(ctx context.Context, msg string) error
	}
)

var (
	localApp IApp
)

func App() IApp {
	if localApp == nil {
		panic("implement not found for interface IApp, forgot register?")
	}
	return localApp
}

func RegisterApp(i IApp) {
	localApp = i
}
