package model

import (
	"github.com/sashabaranov/go-openai"
)

type CompletionsReq struct {
	Messages        []openai.ChatCompletionMessage `json:"messages"`
	Stream          bool                           `json:"stream"`
	Model           string                         `json:"model"`
	Temperature     float64                        `json:"temperature"`
	PresencePenalty int                            `json:"presence_penalty"`
}

type CompletionsRes struct {
	Completion   string       `json:"completion"`
	Usage        openai.Usage `json:"usage"`
	Error        error        `json:"err"`
	ConnTime     int64        `json:"-"`
	Duration     int64        `json:"-"`
	TotalTime    int64        `json:"-"`
	InternalTime int64        `json:"-"`
	EnterTime    int64        `json:"-"`
}
