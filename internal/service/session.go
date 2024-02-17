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
		// 保存应用和密钥是否限制额度
		SaveIsLimitQuota(ctx context.Context, app, key bool) error
		// 获取用户ID
		GetUserId(ctx context.Context) int
		// 获取应用ID
		GetAppId(ctx context.Context) int
		// 获取密钥
		GetSecretKey(ctx context.Context) string
		// 获取应用是否限制额度
		GetAppIsLimitQuota(ctx context.Context) bool
		// 获取密钥是否限制额度
		GetKeyIsLimitQuota(ctx context.Context) bool
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
