package google

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/api/google/v1"
	"github.com/iimeta/fastapi/v2/internal/logic/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func (c *ControllerV1) Interactions(ctx context.Context, req *v1.InteractionsReq) (res *v1.InteractionsRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Google Interactions time: %d", gtime.TimestampMilli()-now)
	}()

	if req.Stream || req.Alt == "sse" {
		if err = service.Google().InteractionsStream(ctx, g.RequestFromCtx(ctx), nil, nil); err != nil {
			return nil, err
		}
		g.RequestFromCtx(ctx).SetCtxVar("stream", true)
	} else {
		response, err := service.Google().Interactions(ctx, g.RequestFromCtx(ctx), nil, nil)
		if err != nil {
			return nil, err
		}

		passthrough, _ := g.RequestFromCtx(ctx).GetCtxVar("passthrough").Val().(*common.EffectivePassthrough)

		// 响应头透传
		common.WritePassthroughHeaders(ctx, passthrough, response.ResponseHeaders)

		g.RequestFromCtx(ctx).Response.WriteJson(response.ResponseBytes)
	}

	return
}
