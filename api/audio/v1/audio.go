package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
)

// Speech接口请求参数
type SpeechReq struct {
	g.Meta `path:"/speech" tags:"audio" method:"post" summary:"Speech接口"`
	smodel.SpeechRequest
}

// Speech接口响应参数
type SpeechRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// Transcriptions接口请求参数
type TranscriptionsReq struct {
	g.Meta `path:"/transcriptions" tags:"audio" method:"post" summary:"Transcriptions接口"`
	smodel.AudioRequest
	Duration float64 `json:"duration"`
}

// Transcriptions接口响应参数
type TranscriptionsRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
