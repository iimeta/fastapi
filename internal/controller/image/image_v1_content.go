package image

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/api/image/v1"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func (c *ControllerV1) Content(ctx context.Context, req *v1.ContentReq) (res *v1.ContentRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Content time: %d", gtime.TimestampMilli()-now)
	}()

	response, err := service.Image().Content(ctx, req)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.ServeContent(fmt.Sprintf("%s_image_%d.png", req.ImageId, req.Index), time.Now(), bytes.NewReader(response.Data))

	return
}
