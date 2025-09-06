package audio

import (
	"bytes"
	"context"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/api/audio/v1"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
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

	g.RequestFromCtx(ctx).Response.ServeContent("speech.mp3", time.Now(), bytes.NewReader(response.Data))

	return
}
