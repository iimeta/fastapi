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
	ILogger interface {
		Chat(ctx context.Context, chatLog model.ChatLog, retry ...int)
	}
)

var (
	localLogger ILogger
)

func Logger() ILogger {
	if localLogger == nil {
		panic("implement not found for interface ILogger, forgot register?")
	}
	return localLogger
}

func RegisterLogger(i ILogger) {
	localLogger = i
}
