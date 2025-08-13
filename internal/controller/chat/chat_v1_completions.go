package chat

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"

	"github.com/iimeta/fastapi/api/chat/v1"
)

func (c *ControllerV1) Completions(ctx context.Context, req *v1.CompletionsReq) (res *v1.CompletionsRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Completions time: %d", gtime.TimestampMilli()-now)
	}()

	if !req.IsToResponses {
		if req.Stream {
			if err = service.Chat().CompletionsStream(ctx, g.RequestFromCtx(ctx).GetBody(), nil, nil); err != nil {
				return nil, err
			}
			g.RequestFromCtx(ctx).SetCtxVar("stream", req.Stream)
		} else {
			response, err := service.Chat().Completions(ctx, g.RequestFromCtx(ctx).GetBody(), nil, nil)
			if err != nil {
				return nil, err
			}
			g.RequestFromCtx(ctx).Response.WriteJson(response)
		}
	} else {
		if req.Stream {
			if err = service.OpenAI().ResponsesStream(ctx, g.RequestFromCtx(ctx), true, nil, nil); err != nil {
				return nil, err
			}
			g.RequestFromCtx(ctx).SetCtxVar("stream", true)
		} else {
			response, err := service.OpenAI().Responses(ctx, g.RequestFromCtx(ctx), true, nil, nil)
			if err != nil {
				return nil, err
			}
			g.RequestFromCtx(ctx).Response.WriteJson(response.ResponseBytes)
		}
	}

	return
}
