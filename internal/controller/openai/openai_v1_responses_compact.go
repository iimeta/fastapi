package openai

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/api/openai/v1"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func (c *ControllerV1) ResponsesCompact(ctx context.Context, req *v1.ResponsesCompactReq) (res *v1.ResponsesCompactRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller OpenAI ResponsesCompact time: %d", gtime.TimestampMilli()-now)
	}()

	response, err := service.OpenAI().ResponsesCompact(ctx, g.RequestFromCtx(ctx), false, nil, nil)
	if err != nil {
		return nil, err
	}
	g.RequestFromCtx(ctx).Response.WriteJson(response.ResponseBytes)

	return
}
