package google

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/api/google/v1"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func (c *ControllerV1) Completions(ctx context.Context, req *v1.CompletionsReq) (res *v1.CompletionsRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Google Completions time: %d", gtime.TimestampMilli()-now)
	}()

	if req.Action == "streamGenerateContent" || req.Alt == "sse" {
		if err = service.Google().CompletionsStream(ctx, g.RequestFromCtx(ctx), nil, nil); err != nil {
			return nil, err
		}
		g.RequestFromCtx(ctx).SetCtxVar("stream", true)
	} else {
		response, err := service.Google().Completions(ctx, g.RequestFromCtx(ctx), nil, nil)
		if err != nil {
			return nil, err
		}
		g.RequestFromCtx(ctx).Response.WriteJson(response.ResponseBytes)
	}

	return
}
