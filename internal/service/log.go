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
	ILog interface {
		// 文本日志
		Text(ctx context.Context, textLog model.LogText, retry ...int)
		// 绘图日志
		Image(ctx context.Context, imageLog model.LogImage, retry ...int)
		// 音频日志
		Audio(ctx context.Context, audioLog model.LogAudio, retry ...int)
		// 视频日志
		Video(ctx context.Context, videoLog model.LogVideo, retry ...int)
		// Midjourney日志
		Midjourney(ctx context.Context, midjourneyLog model.LogMidjourney, retry ...int)
	}
)

var (
	localLog ILog
)

func Log() ILog {
	if localLog == nil {
		panic("implement not found for interface ILog, forgot register?")
	}
	return localLog
}

func RegisterLog(i ILog) {
	localLog = i
}
