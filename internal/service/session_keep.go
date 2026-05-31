// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	"github.com/iimeta/fastapi/v2/internal/model/common"
)

type (
	ISessionKeepModelAgent interface {
		// 获取会话保持绑定的代理和密钥
		Get(ctx context.Context, sk *common.SessionKey) (string, string, bool, error)
		// 设置会话保持绑定
		Set(ctx context.Context, sk *common.SessionKey, agentId string, keyId string) error
		// 刷新会话保持绑定
		Refresh(ctx context.Context, sk *common.SessionKey, agentId string, keyId string) error
		// 删除会话保持绑定
		Delete(ctx context.Context, sk *common.SessionKey, agentId string) error
		// 根据代理删除所有会话保持绑定
		DeleteByAgent(ctx context.Context, agentId string) (int64, error)
		// 删除所有会话保持绑定
		DeleteAll(ctx context.Context) (int64, error)
		// 记录代理失败次数
		RecordFail(ctx context.Context, sk *common.SessionKey, agentId string) (int64, error)
		// 清除代理失败计数
		ClearFail(ctx context.Context, sk *common.SessionKey, agentId string) error
		// 记录密钥失败次数
		RecordKeyFail(ctx context.Context, sk *common.SessionKey, agentId string, keyId string) (int64, error)
		// 清除密钥失败计数
		ClearKeyFail(ctx context.Context, sk *common.SessionKey, agentId string, keyId string) error
		// 解析会话保持Key
		ResolveSessionKey(ctx context.Context, modelName string, cfg *common.ModelAgentSessionKeep) *common.SessionKey
	}
)

var (
	localSessionKeepModelAgent ISessionKeepModelAgent
)

func SessionKeepModelAgent() ISessionKeepModelAgent {
	if localSessionKeepModelAgent == nil {
		panic("implement not found for interface ISessionKeepModelAgent, forgot register?")
	}
	return localSessionKeepModelAgent
}

func RegisterSessionKeepModelAgent(i ISessionKeepModelAgent) {
	localSessionKeepModelAgent = i
}
