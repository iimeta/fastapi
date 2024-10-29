package file

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"

	"github.com/iimeta/fastapi/api/file/v1"
)

func (c *ControllerV1) Files(ctx context.Context, req *v1.FilesReq) (res *v1.FilesRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Transcriptions time: %d", gtime.TimestampMilli()-now)
	}()

	fileName, err := req.File.Save("./resource/file/", true)
	if err != nil {
		return nil, err
	}

	req.FileFilesReq.FilePath = "./resource/file/" + fileName

	response, err := service.File().Files(ctx, req.FileFilesReq)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.WriteJson(response)

	return
}
