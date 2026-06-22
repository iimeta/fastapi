package image

import (
	"context"
	"slices"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/api/image/v1"
	"github.com/iimeta/fastapi/v2/internal/logic/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func (c *ControllerV1) Edits(ctx context.Context, req *v1.EditsReq) (res *v1.EditsRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Edits time: %d", gtime.TimestampMilli()-now)
	}()

	request := g.RequestFromCtx(ctx)

	if request != nil && request.MultipartForm != nil && request.MultipartForm.File != nil {
		req.Image = request.GetMultipartFiles("image")
		if fhs := request.MultipartForm.File["mask"]; len(fhs) > 0 {
			req.Mask = fhs[0]
		}
	}

	if req.Async {

		response, err := service.Image().EditsAsync(ctx, req.ImageEditRequest, nil, nil)
		if err != nil {
			return nil, err
		}

		g.RequestFromCtx(ctx).Response.WriteJson(response)

	} else if req.Stream {

		if err = service.Image().EditsStream(ctx, req.ImageEditRequest, nil, nil); err != nil {
			return nil, err
		}

		g.RequestFromCtx(ctx).SetCtxVar("stream", req.Stream)

	} else {

		response, err := service.Image().Edits(ctx, req.ImageEditRequest, nil, nil)
		if err != nil {
			return nil, err
		}

		passthrough, _ := g.RequestFromCtx(ctx).GetCtxVar("passthrough").Val().(*common.EffectivePassthrough)
		isResDataPassthrough := passthrough != nil && slices.Contains(passthrough.ResParams, "res_data")

		// 响应头透传
		common.WritePassthroughHeaders(ctx, passthrough, response.ResponseHeaders)

		if !isResDataPassthrough || response.ResponseBytes == nil {
			g.RequestFromCtx(ctx).Response.WriteJson(response)
		} else {
			g.RequestFromCtx(ctx).Response.WriteJson(response.ResponseBytes)
		}
	}

	return
}
