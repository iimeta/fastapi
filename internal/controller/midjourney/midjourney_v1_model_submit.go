package midjourney

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"

	"github.com/iimeta/fastapi/api/midjourney/v1"
)

func (c *ControllerV1) ModelSubmit(ctx context.Context, req *v1.ModelSubmitReq) (res *v1.ModelSubmitRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Midjourney ModelSubmit time: %d", gtime.TimestampMilli()-now)
	}()

	response, err := service.Midjourney().Submit(ctx, g.RequestFromCtx(ctx))
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.WriteJson(response.Response)

	return
}
