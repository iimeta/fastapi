package midjourney

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/api/midjourney/v1"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func (c *ControllerV1) ModelSubmit(ctx context.Context, req *v1.ModelSubmitReq) (res *v1.ModelSubmitRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Midjourney ModelSubmit time: %d", gtime.TimestampMilli()-now)
	}()

	response, err := service.Midjourney().Submit(ctx, g.RequestFromCtx(ctx), nil, nil)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.WriteJson(response.Response)

	return
}
