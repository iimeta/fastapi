package batch

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/api/batch/v1"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func (c *ControllerV1) Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Create time: %d", gtime.TimestampMilli()-now)
	}()

	response, err := service.Batch().Create(ctx, req, nil, nil)
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
