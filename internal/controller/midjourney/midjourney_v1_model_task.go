package midjourney

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/api/midjourney/v1"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

func (c *ControllerV1) ModelTask(ctx context.Context, req *v1.ModelTaskReq) (res *v1.ModelTaskRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Midjourney ModelTask time: %d", gtime.TimestampMilli()-now)
	}()

	response, err := service.Midjourney().Task(ctx, g.RequestFromCtx(ctx), nil, nil)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.WriteJson(response.Response)

	return
}
