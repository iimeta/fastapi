package volcengine

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/api/volcengine/v1"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func (c *ControllerV1) VideoList(ctx context.Context, req *v1.VideoListReq) (res *v1.VideoListRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller VolcEngine VideoList time: %d", gtime.TimestampMilli()-now)
	}()

	responseBytes, err := service.VolcEngine().VideoList(ctx, g.RequestFromCtx(ctx), nil, nil)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.WriteJson(responseBytes)

	return
}
