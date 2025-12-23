package file

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/api/file/v1"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

func (c *ControllerV1) Upload(ctx context.Context, req *v1.UploadReq) (res *v1.UploadRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Upload time: %d", gtime.TimestampMilli()-now)
	}()

	_, fileHeader, err := g.RequestFromCtx(ctx).FormFile("file")
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	req.File = fileHeader

	if req.Model == "" {
		req.Model = req.Provider + "-files"
	}

	response, err := service.File().Upload(ctx, req, nil, nil)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.WriteJson(response)

	return
}
