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
		// 聊天日志
		Chat(ctx context.Context, chatLog model.ChatLog, retry ...int)
		// 绘图日志
		Image(ctx context.Context, imageLog model.ImageLog, retry ...int)
		// 音频日志
		Audio(ctx context.Context, audioLog model.AudioLog, retry ...int)
		// Midjourney日志
		Midjourney(ctx context.Context, midjourneyLog model.MidjourneyLog, retry ...int)
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
