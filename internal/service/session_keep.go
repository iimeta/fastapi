// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
)

type (
	ISessionKeepModelAgent interface {
		Get(ctx context.Context, userId int, modelName string) (string, bool, error)
		Set(ctx context.Context, userId int, modelName string, agentId string) error
		Refresh(ctx context.Context, userId int, modelName string, agentId string) error
		Delete(ctx context.Context, userId int, modelName string, agentId string) error
		DeleteByAgent(ctx context.Context, agentId string) (int64, error)
		DeleteAll(ctx context.Context) (int64, error)
		RecordFail(ctx context.Context, userId int, modelName string, agentId string) (int64, error)
		ClearFail(ctx context.Context, userId int, modelName string, agentId string) error
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
