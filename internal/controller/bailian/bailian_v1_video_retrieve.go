package bailian

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/api/bailian/v1"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func (c *ControllerV1) VideoRetrieve(ctx context.Context, req *v1.VideoRetrieveReq) (res *v1.VideoRetrieveRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Bailian VideoRetrieve time: %d", gtime.TimestampMilli()-now)
	}()

	responseBytes, err := service.Bailian().VideoRetrieve(ctx, g.RequestFromCtx(ctx), req.TaskId, nil, nil)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.WriteJson(responseBytes)

	return
}
