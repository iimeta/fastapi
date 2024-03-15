package midjourney

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"

	"github.com/iimeta/fastapi/api/midjourney/v1"
)

func (c *ControllerV1) Action(ctx context.Context, req *v1.ActionReq) (res *v1.ActionRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Action time: %d", gtime.TimestampMilli()-now)
	}()

	response, err := service.Midjourney().Action(ctx, req.MidjourneyProxyRequest)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.WriteJson(response)

	return
}
