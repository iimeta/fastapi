package file

import (
	"bufio"
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi/v2/api/file/v1"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"github.com/iimeta/fastapi/v2/utility/util"
)

func (c *ControllerV1) Upload(ctx context.Context, req *v1.UploadReq) (res *v1.UploadRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Upload time: %d", gtime.TimestampMilli()-now)
	}()

	file, fileHeader, err := g.RequestFromCtx(ctx).FormFile("file")
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	defer func() {
		if err := file.Close(); err != nil {
			logger.Error(ctx, err)
		}
	}()

	if req.Model == "" && req.Purpose == "batch" {

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if body, ok := util.ConvToMap(scanner.Bytes())["body"]; ok {
				if model, ok := util.ConvToMap(body)["model"]; ok {
					req.Model = gconv.String(model)
					break
				}
			}
		}
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
