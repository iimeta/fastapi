package bailian

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/iimeta/fastapi/v2/api/bailian/v1"
	"github.com/iimeta/fastapi/v2/internal/service"
)

func (c *ControllerV1) ImageGenerations(ctx context.Context, req *v1.ImageGenerationsReq) (res *v1.ImageGenerationsRes, err error) {

	if req.Async {

		response, err := service.Bailian().ImageGenerationsAsync(ctx, g.RequestFromCtx(ctx).GetBody(), nil, nil)
		if err != nil {
			return nil, err
		}

		g.RequestFromCtx(ctx).Response.WriteJson(response)

		return nil, nil
	}

	responseBytes, err := service.Bailian().ImageGenerations(ctx, g.RequestFromCtx(ctx).GetBody(), nil, nil)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.WriteJson(responseBytes)

	return nil, nil
}
