package service

import (
	"context"

	"github.com/iimeta/fastapi/v2/internal/model/common"
)

type (
	ISessionKeepModelAgent interface {
		ResolveSessionKey(ctx context.Context, modelName string, cfg *common.ModelAgentSessionKeep) *common.SessionKey
		Get(ctx context.Context, sk *common.SessionKey) (agentId string, keyId string, ok bool, err error)
		Set(ctx context.Context, sk *common.SessionKey, agentId string, keyId string) error
		Refresh(ctx context.Context, sk *common.SessionKey, agentId string, keyId string) error
		Delete(ctx context.Context, sk *common.SessionKey, agentId string) error
		DeleteByAgent(ctx context.Context, agentId string) (int64, error)
		DeleteAll(ctx context.Context) (int64, error)
		RecordFail(ctx context.Context, sk *common.SessionKey, agentId string) (int64, error)
		ClearFail(ctx context.Context, sk *common.SessionKey, agentId string) error
		RecordKeyFail(ctx context.Context, sk *common.SessionKey, agentId string, keyId string) (int64, error)
		ClearKeyFail(ctx context.Context, sk *common.SessionKey, agentId string, keyId string) error
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
