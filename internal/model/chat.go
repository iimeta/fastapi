package model

import "github.com/sashabaranov/go-openai"

type CompletionsReq struct {
	Messages        []openai.ChatCompletionMessage `json:"messages"`
	Stream          bool                           `json:"stream"`
	Model           string                         `json:"model"`
	Temperature     float64                        `json:"temperature"`
	PresencePenalty int                            `json:"presence_penalty"`
}
