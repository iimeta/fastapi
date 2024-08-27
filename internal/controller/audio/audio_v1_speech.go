package audio

import (
	"bytes"
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"io"
	"time"

	"github.com/iimeta/fastapi/api/audio/v1"
)

func (c *ControllerV1) Speech(ctx context.Context, req *v1.SpeechReq) (res *v1.SpeechRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Speech time: %d", gtime.TimestampMilli()-now)
	}()

	response, err := service.Audio().Speech(ctx, req.SpeechRequest, nil)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(response.ReadCloser)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.ServeContent("speech.mp3", time.Now(), bytes.NewReader(data))

	return
}
