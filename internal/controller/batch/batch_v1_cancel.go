package batch

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/api/batch/v1"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func (c *ControllerV1) Cancel(ctx context.Context, req *v1.CancelReq) (res *v1.CancelRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Cancel time: %d", gtime.TimestampMilli()-now)
	}()

	response, err := service.Batch().Cancel(ctx, req)
	if err != nil {
		return nil, err
	}

	if response.ResponseBytes == nil {
		g.RequestFromCtx(ctx).Response.WriteJson(response)
	} else {
		g.RequestFromCtx(ctx).Response.WriteJson(response.ResponseBytes)
	}

	return
}
