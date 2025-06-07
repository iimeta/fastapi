// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	sdkm "github.com/iimeta/fastapi-sdk/model"
	v1 "github.com/iimeta/fastapi/api/audio/v1"
	"github.com/iimeta/fastapi/internal/model"
)

type (
	IAudio interface {
		// Speech
		Speech(ctx context.Context, params sdkm.SpeechRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.SpeechResponse, err error)
		// Transcriptions
		Transcriptions(ctx context.Context, params *v1.TranscriptionsReq, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response sdkm.AudioResponse, err error)
		// 保存日志
		SaveLog(ctx context.Context, audioLog model.AudioLog, retry ...int)
	}
)

var (
	localAudio IAudio
)

func Audio() IAudio {
	if localAudio == nil {
		panic("implement not found for interface IAudio, forgot register?")
	}
	return localAudio
}

func RegisterAudio(i IAudio) {
	localAudio = i
}
