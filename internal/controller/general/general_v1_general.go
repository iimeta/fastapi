package general

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/api/general/v1"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

func (c *ControllerV1) General(ctx context.Context, req *v1.GeneralReq) (res *v1.GeneralRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller General time: %d", gtime.TimestampMilli()-now)
	}()

	if req.Stream {
		if err = service.General().GeneralStream(ctx, g.RequestFromCtx(ctx), nil, nil); err != nil {
			return nil, err
		}
		g.RequestFromCtx(ctx).SetCtxVar("stream", req.Stream)
	} else {

		response, err := service.General().General(ctx, g.RequestFromCtx(ctx), nil, nil)
		if err != nil {
			return nil, err
		}

		if response.ResponseBytes != nil {
			g.RequestFromCtx(ctx).Response.WriteJson(response.ResponseBytes)
		} else {
			g.RequestFromCtx(ctx).Response.WriteJson(response)
		}
	}

	return
}
