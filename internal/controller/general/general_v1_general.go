package general

import (
	"context"
	"slices"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/api/general/v1"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/internal/errors"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func (c *ControllerV1) General(ctx context.Context, req *v1.GeneralReq) (res *v1.GeneralRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller General time: %d", gtime.TimestampMilli()-now)
	}()

	r := g.RequestFromCtx(ctx)
	if r == nil {
		return
	}

	if !config.Cfg.GeneralApi.Open || (len(config.Cfg.GeneralApi.IpWhitelist) > 0 && !slices.Contains(config.Cfg.GeneralApi.IpWhitelist, r.GetClientIp())) {
		return nil, errors.ERR_NOT_FOUND
	}

	if req.Stream {
		if err = service.General().GeneralStream(ctx, r, nil, nil); err != nil {
			return nil, err
		}
		r.SetCtxVar("stream", req.Stream)
	} else {

		response, err := service.General().General(ctx, r, nil, nil)
		if err != nil {
			return nil, err
		}

		if response.ResponseBytes != nil {
			r.Response.WriteJson(response.ResponseBytes)
		} else {
			r.Response.WriteJson(response)
		}
	}

	return
}
