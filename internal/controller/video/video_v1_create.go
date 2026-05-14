package video

import (
	"context"
	"slices"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/api/video/v1"
	"github.com/iimeta/fastapi/v2/internal/logic/common"
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

	passthrough, _ := g.RequestFromCtx(ctx).GetCtxVar("passthrough").Val().(*common.EffectivePassthrough)
	isResDataPassthrough := passthrough != nil && slices.Contains(passthrough.ResParams, "res_data")

	// 响应头透传
	common.WritePassthroughHeaders(ctx, passthrough, response.ResponseHeaders)

	if !isResDataPassthrough || response.ResponseBytes == nil {
		g.RequestFromCtx(ctx).Response.WriteJson(response)
	} else {
		g.RequestFromCtx(ctx).Response.WriteJson(response.ResponseBytes)
	}

	return
}
