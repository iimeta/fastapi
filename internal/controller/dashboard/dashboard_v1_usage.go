package dashboard

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/iimeta/fastapi/v2/api/dashboard/v1"
	"github.com/iimeta/fastapi/v2/internal/service"
)

func (c *ControllerV1) Usage(ctx context.Context, req *v1.UsageReq) (res *v1.UsageRes, err error) {

	usage, err := service.Dashboard().Usage(ctx)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.WriteJson(usage)

	return
}
