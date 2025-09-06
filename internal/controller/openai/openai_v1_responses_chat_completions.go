package openai

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/api/openai/v1"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

func (c *ControllerV1) ResponsesChatCompletions(ctx context.Context, req *v1.ResponsesChatCompletionsReq) (res *v1.ResponsesChatCompletionsRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller ResponsesChatCompletions time: %d", gtime.TimestampMilli()-now)
	}()

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

	return
}
