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
	IUser interface {
		// 根据用户ID获取用户信息
		GetUser(ctx context.Context, userId int) (*model.User, error)
		// 用户列表
		List(ctx context.Context) ([]*model.User, error)
		// 用户消费额度
		SpendQuota(ctx context.Context, userId, quota, currentQuota int) error
		// 保存用户信息到缓存
		SaveCacheUser(ctx context.Context, user *model.User) error
		// 获取缓存中的用户信息
		GetCacheUser(ctx context.Context, userId int) (*model.User, error)
		// 更新缓存中的用户信息
		UpdateCacheUser(ctx context.Context, user *entity.User)
		// 移除缓存中的用户信息
		RemoveCacheUser(ctx context.Context, userId int)
		// 变更订阅
		Subscribe(ctx context.Context, msg string) error
	}
)

var (
	localUser IUser
)

func User() IUser {
	if localUser == nil {
		panic("implement not found for interface IUser, forgot register?")
	}
	return localUser
}

func RegisterUser(i IUser) {
	localUser = i
}
