package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	sdkm "github.com/iimeta/fastapi-sdk/model"
)

// Speech接口请求参数
type SpeechReq struct {
	g.Meta `path:"/speech" tags:"audio" method:"post" summary:"audio接口"`
	sdkm.SpeechRequest
}

// Speech接口响应参数
type SpeechRes struct {
	g.Meta `mime:"application/json" example:"json"`
}

// Transcriptions接口请求参数
type TranscriptionsReq struct {
	g.Meta `path:"/transcriptions" tags:"audio" method:"post" summary:"transcriptions接口"`
	sdkm.AudioRequest
	Duration float64 `json:"duration"`
}

// Transcriptions接口响应参数
type TranscriptionsRes struct {
	g.Meta `mime:"application/json" example:"json"`
}
