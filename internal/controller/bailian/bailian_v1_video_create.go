package bailian

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/api/bailian/v1"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func (c *ControllerV1) VideoCreate(ctx context.Context, req *v1.VideoCreateReq) (res *v1.VideoCreateRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Bailian VideoCreate time: %d", gtime.TimestampMilli()-now)
	}()

	responseBytes, err := service.Bailian().VideoCreate(ctx, g.RequestFromCtx(ctx), nil, nil)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.WriteJson(responseBytes)

	return
}
