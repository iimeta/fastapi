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
	ISession interface {
		// 保存会话
		Save(ctx context.Context, secretKey string) error
		// 保存应用和密钥是否限制额度
		SaveIsLimitQuota(ctx context.Context, app bool, key bool)
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
		// 保存用户信息到会话中
		SaveUser(ctx context.Context, user *model.User)
		// 获取会话中的用户信息
		GetUser(ctx context.Context) *model.User
		// 保存应用信息到会话中
		SaveApp(ctx context.Context, app *model.App)
		// 获取会话中的应用信息
		GetApp(ctx context.Context) *model.App
		// 保存密钥信息到会话中
		SaveKey(ctx context.Context, key *model.Key)
		// 获取会话中的密钥信息
		GetKey(ctx context.Context) *model.Key
		// 记录错误模型代理ID到会话中
		RecordErrorModelAgent(ctx context.Context, id string)
		// 获取会话中的错误模型代理Ids
		GetErrorModelAgents(ctx context.Context) []string
		// 记录错误密钥ID到会话中
		RecordErrorKey(ctx context.Context, id string)
		// 获取会话中的错误密钥Ids
		GetErrorKeys(ctx context.Context) []string
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
