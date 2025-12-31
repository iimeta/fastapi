package audio

import (
	"context"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/api/audio/v1"
	"github.com/iimeta/fastapi/v2/internal/errors"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"github.com/iimeta/fastapi/v2/utility/util"
)

func (c *ControllerV1) Transcriptions(ctx context.Context, req *v1.TranscriptionsReq) (res *v1.TranscriptionsRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Transcriptions time: %d", gtime.TimestampMilli()-now)
	}()

	request := g.RequestFromCtx(ctx)

	file, fileHeader, err := request.FormFile("file")
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	req.File = fileHeader

	if req.ResponseFormat != "verbose_json" {

		duration, err := util.GetAudioDuration(file, fileHeader.Filename)
		if err != nil {
			logger.Error(ctx, err)
			return nil, err
		}

		req.Duration = duration.Seconds()
		if req.Duration == 0 {
			logger.Errorf(ctx, "req: %s, error: %v", gjson.MustEncodeString(req), errors.ERR_UNSUPPORTED_FILE_FORMAT)
			return nil, errors.ERR_UNSUPPORTED_FILE_FORMAT
		} else if req.Duration < 1 {
			req.Duration = 1
		}
	}

	response, err := service.Audio().Transcriptions(ctx, req, nil, nil)
	if err != nil {
		return nil, err
	}

	if req.ResponseFormat == "" || req.ResponseFormat == "json" || req.ResponseFormat == "verbose_json" {
		g.RequestFromCtx(ctx).Response.WriteJson(response)
	} else {
		g.RequestFromCtx(ctx).Response.Write(response.Text)
	}

	return
}
