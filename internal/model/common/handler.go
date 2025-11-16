package common

import (
	smodel "github.com/iimeta/fastapi-sdk/model"
)

type BeforeHandler struct {
}

type AfterHandler struct {
	ChatCompletionReq      smodel.ChatCompletionRequest
	ImageGenerationRequest smodel.ImageGenerationRequest
	ImageEditRequest       smodel.ImageEditRequest
	ImageResponse          smodel.ImageResponse
	AudioInput             string
	AudioMinute            float64
	AudioText              string
	EmbeddingReq           smodel.EmbeddingRequest
	ModerationReq          smodel.ModerationRequest
	ChatCompletionRes      smodel.ChatCompletionResponse
	Completion             string
	ServiceTier            string
	MidjourneyPath         string
	MidjourneyResponse     smodel.MidjourneyResponse
	MidjourneyReqUrl       string
	MidjourneyTaskId       string
	MidjourneyPrompt       string
	Usage                  *smodel.Usage
	Error                  error
	RetryInfo              *Retry
	Spend                  Spend
	IsSmartMatch           bool
	ConnTime               int64
	Duration               int64
	TotalTime              int64
	InternalTime           int64
	EnterTime              int64
}
