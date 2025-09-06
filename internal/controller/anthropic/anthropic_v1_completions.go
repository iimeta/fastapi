package anthropic

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/api/anthropic/v1"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

func (c *ControllerV1) Completions(ctx context.Context, req *v1.CompletionsReq) (res *v1.CompletionsRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Anthropic Completions time: %d", gtime.TimestampMilli()-now)
	}()

	if req.Stream {
		if err = service.Anthropic().CompletionsStream(ctx, g.RequestFromCtx(ctx), nil, nil); err != nil {
			return nil, err
		}
		g.RequestFromCtx(ctx).SetCtxVar("stream", true)
	} else {
		response, err := service.Anthropic().Completions(ctx, g.RequestFromCtx(ctx), nil, nil)
		if err != nil {
			return nil, err
		}
		g.RequestFromCtx(ctx).Response.WriteJson(response.ResponseBytes)
	}

	return
}
