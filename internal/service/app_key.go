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
	IAppKey interface {
		// 根据secretKey获取应用密钥信息
		GetBySecretKey(ctx context.Context, secretKey string) (*model.AppKey, error)
		// 应用密钥列表
		List(ctx context.Context) ([]*model.AppKey, error)
		// 保存应用密钥信息到缓存
		SaveCache(ctx context.Context, key *model.AppKey) error
		// 获取缓存中的应用密钥信息
		GetCache(ctx context.Context, secretKey string) (*model.AppKey, error)
		// 更新缓存中的应用密钥信息
		UpdateCache(ctx context.Context, key *entity.AppKey)
		// 移除缓存中的应用密钥信息
		RemoveCache(ctx context.Context, secretKey string)
		// 应用密钥花费额度
		SpendQuota(ctx context.Context, secretKey string, spendQuota int, currentQuota int) error
		// 应用密钥已用额度
		UsedQuota(ctx context.Context, secretKey string, quota int) error
		// 保存应用密钥额度到缓存
		SaveCacheQuota(ctx context.Context, secretKey string, quota int) error
		// 获取缓存中的应用密钥额度
		GetCacheQuota(ctx context.Context, secretKey string) int
		// 更新应用密钥额度过期时间
		UpdateQuotaExpiresAt(ctx context.Context, key *model.AppKey) error
		// 变更订阅
		Subscribe(ctx context.Context, msg string) error
	}
)

var (
	localAppKey IAppKey
)

func AppKey() IAppKey {
	if localAppKey == nil {
		panic("implement not found for interface IAppKey, forgot register?")
	}
	return localAppKey
}

func RegisterAppKey(i IAppKey) {
	localAppKey = i
}
