package video

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/api/video/v1"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func (c *ControllerV1) Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Create time: %d", gtime.TimestampMilli()-now)
	}()

	if _, fileHeader, err := g.RequestFromCtx(ctx).FormFile("input_reference"); err == nil {
		req.InputReference = fileHeader
	}

	response, err := service.Video().Create(ctx, req, nil, nil)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.WriteJson(response)

	return
}
