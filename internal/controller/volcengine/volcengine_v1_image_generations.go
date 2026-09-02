package volcengine

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/iimeta/fastapi/v2/api/volcengine/v1"
	"github.com/iimeta/fastapi/v2/internal/service"
)

func (c *ControllerV1) ImageGenerations(ctx context.Context, req *v1.ImageGenerationsReq) (res *v1.ImageGenerationsRes, err error) {

	if req.Async {

		response, err := service.Image().GenerationsAsync(ctx, g.RequestFromCtx(ctx).GetBody(), nil, nil)
		if err != nil {
			return nil, err
		}

		g.RequestFromCtx(ctx).Response.WriteJson(response)

		return nil, nil
	}

	if req.Stream {

		if err = service.VolcEngine().ImageGenerationsStream(ctx, g.RequestFromCtx(ctx).GetBody(), nil, nil); err != nil {
			return nil, err
		}

		g.RequestFromCtx(ctx).SetCtxVar("stream", req.Stream)

		return nil, nil
	}

	responseBytes, err := service.VolcEngine().ImageGenerations(ctx, g.RequestFromCtx(ctx).GetBody(), nil, nil)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.WriteJson(responseBytes)

	return nil, nil
}
