package model

import (
	"github.com/sashabaranov/go-openai"
)

type ImageReq struct {
	Prompt         string `json:"prompt,omitempty"`
	Model          string `json:"model,omitempty"`
	N              int    `json:"n,omitempty"`
	Quality        string `json:"quality,omitempty"`
	Size           string `json:"size,omitempty"`
	Style          string `json:"style,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	User           string `json:"user,omitempty"`
}

type ImageRes struct {
	Created      int64                           `json:"created,omitempty"`
	Data         []openai.ImageResponseDataInner `json:"data,omitempty"`
	Usage        *openai.Usage                   `json:"usage"`
	Error        error                           `json:"err"`
	ConnTime     int64                           `json:"-"`
	Duration     int64                           `json:"-"`
	TotalTime    int64                           `json:"-"`
	InternalTime int64                           `json:"-"`
	EnterTime    int64                           `json:"-"`
}
