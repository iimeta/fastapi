package moderation

import (
	"context"
	"slices"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/api/moderation/v1"
	"github.com/iimeta/fastapi/v2/internal/logic/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
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
