package midjourney

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"

	"github.com/iimeta/fastapi/api/midjourney/v1"
)

func (c *ControllerV1) Submit(ctx context.Context, req *v1.SubmitReq) (res *v1.SubmitRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Midjourney Submit time: %d", gtime.TimestampMilli()-now)
	}()

	response, err := service.Midjourney().Submit(ctx, g.RequestFromCtx(ctx), nil)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.WriteJson(response.Response)

	return
}
