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
		GetUserByUid(ctx context.Context, userId int) (*model.User, error)
		// 用户列表
		List(ctx context.Context) ([]*model.User, error)
		// 根据用户ID更新额度
		UpdateQuota(ctx context.Context, userId, quota int) (err error)
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
