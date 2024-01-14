// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
)

type (
	ISession interface {
		// 保存会话
		Save(ctx context.Context, secretKey string) error
		// 获取用户ID
		GetUserId(ctx context.Context) int
		// 获取应用ID
		GetAppId(ctx context.Context) int
		// 获取密钥
		GetSecretKey(ctx context.Context) string
	}
)

var (
	localSession ISession
)

func Session() ISession {
	if localSession == nil {
		panic("implement not found for interface ISession, forgot register?")
	}
	return localSession
}

func RegisterSession(i ISession) {
	localSession = i
}
