package image

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"

	"github.com/iimeta/fastapi/api/image/v1"
)

func (c *ControllerV1) Generations(ctx context.Context, req *v1.GenerationsReq) (res *v1.GenerationsRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Generations time: %d", gtime.TimestampMilli()-now)
	}()

	response, err := service.Image().Generations(ctx, req.ImageRequest, nil, nil)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.WriteJson(response)

	return
}
