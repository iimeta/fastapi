// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package audio

import (
	"context"

	"github.com/iimeta/fastapi/api/audio/v1"
)

type IAudioV1 interface {
	Speech(ctx context.Context, req *v1.SpeechReq) (res *v1.SpeechRes, err error)
	Transcriptions(ctx context.Context, req *v1.TranscriptionsReq) (res *v1.TranscriptionsRes, err error)
}
