package model

import (
	sdkm "github.com/iimeta/fastapi-sdk/model"
)

type RealtimeRequest struct {
	Model    string                       `json:"model"`
	Messages []sdkm.ChatCompletionMessage `json:"messages"`
}
