// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	v1 "github.com/iimeta/fastapi/v2/api/audio/v1"
	"github.com/iimeta/fastapi/v2/internal/model"
)

type (
	IAudio interface {
		// Speech
		Speech(ctx context.Context, data []byte, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.SpeechResponse, err error)
		// Transcriptions
		Transcriptions(ctx context.Context, params *v1.TranscriptionsReq, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.AudioResponse, err error)
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
