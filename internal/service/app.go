// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	"github.com/iimeta/fastapi/internal/model"
)

type (
	IApp interface {
		// 根据应用ID获取应用信息
		GetApp(ctx context.Context, appId int) (*model.App, error)
		// 应用列表
		List(ctx context.Context) ([]*model.App, error)
		// 更改应用额度
		ChangeQuota(ctx context.Context, appId, quota int) error
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
