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
	IUser interface {
		// 根据userId获取用户信息
		GetUserByUserId(ctx context.Context, userId int) (*model.User, error)
		// 用户列表
		List(ctx context.Context) ([]*model.User, error)
		// 更改用户额度
		ChangeQuota(ctx context.Context, userId, quota int) error
		// 更新订阅
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
