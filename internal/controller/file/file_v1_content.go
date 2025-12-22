package file

import (
	"bytes"
	"context"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/api/file/v1"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

func (c *ControllerV1) Content(ctx context.Context, req *v1.ContentReq) (res *v1.ContentRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Content time: %d", gtime.TimestampMilli()-now)
	}()

	response, err := service.File().Content(ctx, req, nil, nil)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.ServeContent(req.FileId, time.Now(), bytes.NewReader(response.Data))

	return
}
