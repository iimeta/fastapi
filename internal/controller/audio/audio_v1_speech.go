package audio

import (
	"bytes"
	"context"
	"slices"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/gtrace"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/api/audio/v1"
	"github.com/iimeta/fastapi/v2/internal/logic/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func (c *ControllerV1) Speech(ctx context.Context, req *v1.SpeechReq) (res *v1.SpeechRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Speech time: %d", gtime.TimestampMilli()-now)
	}()

	response, err := service.Audio().Speech(ctx, g.RequestFromCtx(ctx).GetBody(), nil, nil)
	if err != nil {
		return nil, err
	}

	passthrough, _ := g.RequestFromCtx(ctx).GetCtxVar("passthrough").Val().(*common.EffectivePassthrough)
	isResDataPassthrough := passthrough != nil && slices.Contains(passthrough.ResParams, "res_data")

	// 响应头透传
	common.WritePassthroughHeaders(ctx, passthrough, response.ResponseHeaders)

	if isResDataPassthrough && response.ResponseBytes != nil {
		g.RequestFromCtx(ctx).Response.Write(response.ResponseBytes)
	} else {
		g.RequestFromCtx(ctx).Response.ServeContent(gtrace.GetTraceID(ctx)+"_speech.mp3", time.Now(), bytes.NewReader(response.Data))
	}

	return
}
