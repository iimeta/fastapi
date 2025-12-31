package common

import (
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
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
	Action                 string
	VideoId                string
	IsFile                 bool
	FileId                 string
	FileRes                smodel.FileResponse
	IsBatch                bool
	BatchId                string
	Prompt                 string
	Seconds                int
	Size                   string
	MidjourneyPath         string
	MidjourneyResponse     smodel.MidjourneyResponse
	MidjourneyReqUrl       string
	MidjourneyTaskId       string
	MidjourneyPrompt       string
	RequestData            map[string]any
	ResponseData           map[string]any
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
