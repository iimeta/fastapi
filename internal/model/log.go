package model

import (
	smodel "github.com/iimeta/fastapi-sdk/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
)

type LogText struct {
	ReqModel           *Model
	RealModel          *Model
	ModelAgent         *ModelAgent
	FallbackModelAgent *ModelAgent
	FallbackModel      *Model
	Key                *Key
	CompletionsReq     *smodel.ChatCompletionRequest
	EmbeddingReq       *smodel.EmbeddingRequest
	ModerationReq      *smodel.ModerationRequest
	CompletionsRes     *CompletionsRes
	RetryInfo          *mcommon.Retry
	Spend              mcommon.Spend
	IsSmartMatch       bool
}

type LogImage struct {
	ReqModel           *Model
	RealModel          *Model
	ModelAgent         *ModelAgent
	FallbackModelAgent *ModelAgent
	FallbackModel      *Model
	Key                *Key
	ImageReq           *smodel.ImageGenerationRequest
	ImageRes           *ImageRes
	RetryInfo          *mcommon.Retry
	Spend              mcommon.Spend
}

type LogAudio struct {
	ReqModel           *Model
	RealModel          *Model
	ModelAgent         *ModelAgent
	FallbackModelAgent *ModelAgent
	FallbackModel      *Model
	Key                *Key
	AudioReq           *AudioReq
	AudioRes           *AudioRes
	RetryInfo          *mcommon.Retry
	Spend              mcommon.Spend
}

type LogMidjourney struct {
	ReqModel           *Model
	RealModel          *Model
	ModelAgent         *ModelAgent
	FallbackModelAgent *ModelAgent
	FallbackModel      *Model
	Key                *Key
	Response           MidjourneyResponse
	RetryInfo          *mcommon.Retry
	Spend              mcommon.Spend
}
