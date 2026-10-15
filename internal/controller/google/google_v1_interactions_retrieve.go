package google

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/api/google/v1"
	"github.com/iimeta/fastapi/v2/internal/logic/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func (c *ControllerV1) InteractionsRetrieve(ctx context.Context, req *v1.InteractionsRetrieveReq) (res *v1.InteractionsRetrieveRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Google InteractionsRetrieve time: %d", gtime.TimestampMilli()-now)
	}()

	response, err := service.Google().InteractionsRetrieve(ctx, req.Id, nil, nil)
	if err != nil {
		return nil, err
	}

	passthrough, _ := g.RequestFromCtx(ctx).GetCtxVar("passthrough").Val().(*common.EffectivePassthrough)

	// 响应头透传
	common.WritePassthroughHeaders(ctx, passthrough, response.ResponseHeaders)

	g.RequestFromCtx(ctx).Response.WriteJson(response.ResponseBytes)

	return
}
