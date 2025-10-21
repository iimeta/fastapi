package common

import (
	smodel "github.com/iimeta/fastapi-sdk/model"
)

type BeforeHandler struct {
}

type AfterHandler struct {
	ChatCompletionReq smodel.ChatCompletionRequest
	EmbeddingReq      smodel.EmbeddingRequest
	ModerationReq     smodel.ModerationRequest
	ChatCompletionRes smodel.ChatCompletionResponse
	Completion        string
	Usage             *smodel.Usage
	Error             error
	RetryInfo         *Retry
	Spend             Spend
	IsSmartMatch      bool
	ConnTime          int64
	Duration          int64
	TotalTime         int64
	InternalTime      int64
	EnterTime         int64
}
