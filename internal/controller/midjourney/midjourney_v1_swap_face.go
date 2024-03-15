package midjourney

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"

	"github.com/iimeta/fastapi/api/midjourney/v1"
)

func (c *ControllerV1) SwapFace(ctx context.Context, req *v1.SwapFaceReq) (res *v1.SwapFaceRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller SwapFace time: %d", gtime.TimestampMilli()-now)
	}()

	response, err := service.Midjourney().SwapFace(ctx, req.MidjourneyProxyRequest)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.WriteJson(response)

	return
}
