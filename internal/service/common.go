// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/sashabaranov/go-openai"
)

type (
	ICommon interface {
		VerifySecretKey(ctx context.Context, secretKey string) error
		RecordUsage(ctx context.Context, model *model.Model, usage openai.Usage) error
		GetUserUsageKey(ctx context.Context) string
		GetAppUsageCountField(ctx context.Context) string
		GetAppUsedTokensField(ctx context.Context) string
		GetAppTotalTokensField(ctx context.Context) string
		GetKeyUsageCountField(ctx context.Context) string
		GetKeyUsedTokensField(ctx context.Context) string
		GetKeyTotalTokensField(ctx context.Context) string
		GetUserUsageCount(ctx context.Context) (int, error)
		GetUserUsedTokens(ctx context.Context) (int, error)
		GetUserTotalTokens(ctx context.Context) (int, error)
		GetAppUsageCount(ctx context.Context) (int, error)
		GetAppUsedTokens(ctx context.Context) (int, error)
		GetAppTotalTokens(ctx context.Context) (int, error)
		GetKeyUsageCount(ctx context.Context) (int, error)
		GetKeyUsedTokens(ctx context.Context) (int, error)
		GetKeyTotalTokens(ctx context.Context) (int, error)
		// 解析密钥
		ParseSecretKey(ctx context.Context, secretKey string) (int, int, error)
		// 保存用户信息到缓存
		SaveCacheUser(ctx context.Context, user *model.User) error
		// 获取缓存中的用户信息
		GetCacheUser(ctx context.Context, userId int) (*model.User, error)
		// 更新缓存中的用户信息
		UpdateCacheUser(ctx context.Context, user *entity.User)
		// 移除缓存中的用户信息
		RemoveCacheUser(ctx context.Context, userId int)
		// 保存应用信息到缓存
		SaveCacheApp(ctx context.Context, app *model.App) error
		// 获取缓存中的应用信息
		GetCacheApp(ctx context.Context, appId int) (*model.App, error)
		// 更新缓存中的应用信息
		UpdateCacheApp(ctx context.Context, app *entity.App)
		// 移除缓存中的应用信息
		RemoveCacheApp(ctx context.Context, appId int)
		// 保存密钥信息到缓存
		SaveCacheKey(ctx context.Context, key *model.Key) error
		// 获取缓存中的密钥信息
		GetCacheKey(ctx context.Context, secretKey string) (*model.Key, error)
		// 更新缓存中的密钥信息
		UpdateCacheKey(ctx context.Context, key *entity.Key)
		// 移除缓存中的密钥信息
		RemoveCacheKey(ctx context.Context, secretKey string)
	}
)

var (
	localCommon ICommon
)

func Common() ICommon {
	if localCommon == nil {
		panic("implement not found for interface ICommon, forgot register?")
	}
	return localCommon
}

func RegisterCommon(i ICommon) {
	localCommon = i
}
