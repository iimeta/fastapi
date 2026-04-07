package volcengine

import (
	"context"
	"net/http"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/api/volcengine/v1"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func (c *ControllerV1) VideoDelete(ctx context.Context, req *v1.VideoDeleteReq) (res *v1.VideoDeleteRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller VolcEngine VideoDelete time: %d", gtime.TimestampMilli()-now)
	}()

	err = service.VolcEngine().VideoDelete(ctx, g.RequestFromCtx(ctx), req.TaskId, nil, nil)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.WriteStatus(http.StatusOK)

	return
}
