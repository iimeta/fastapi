package moderation

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"

	"github.com/iimeta/fastapi/api/moderation/v1"
)

func (c *ControllerV1) Moderations(ctx context.Context, req *v1.ModerationsReq) (res *v1.ModerationsRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Moderations time: %d", gtime.TimestampMilli()-now)
	}()

	response, err := service.Moderation().Moderations(ctx, req.ModerationRequest, nil, nil)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.WriteJson(response)

	return
}
