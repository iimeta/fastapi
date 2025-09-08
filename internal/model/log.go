package model

import (
	smodel "github.com/iimeta/fastapi-sdk/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
)

type ChatLog struct {
	Group              *Group
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
	IsSmartMatch       bool
}

type ImageLog struct {
	Group              *Group
	ReqModel           *Model
	RealModel          *Model
	ModelAgent         *ModelAgent
	FallbackModelAgent *ModelAgent
	FallbackModel      *Model
	Key                *Key
	ImageReq           *smodel.ImageGenerationRequest
	ImageRes           *ImageRes
	RetryInfo          *mcommon.Retry
}

type AudioLog struct {
	Group              *Group
	ReqModel           *Model
	RealModel          *Model
	ModelAgent         *ModelAgent
	FallbackModelAgent *ModelAgent
	FallbackModel      *Model
	Key                *Key
	AudioReq           *AudioReq
	AudioRes           *AudioRes
	RetryInfo          *mcommon.Retry
}

type MidjourneyLog struct {
	Group              *Group
	ReqModel           *Model
	RealModel          *Model
	ModelAgent         *ModelAgent
	FallbackModelAgent *ModelAgent
	FallbackModel      *Model
	Key                *Key
	Response           MidjourneyResponse
	RetryInfo          *mcommon.Retry
}
