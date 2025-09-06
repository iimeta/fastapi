package embedding

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/api/embedding/v1"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

func (c *ControllerV1) Embeddings(ctx context.Context, req *v1.EmbeddingsReq) (res *v1.EmbeddingsRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Embeddings time: %d", gtime.TimestampMilli()-now)
	}()

	response, err := service.Embedding().Embeddings(ctx, g.RequestFromCtx(ctx).GetBody(), nil, nil)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.WriteJson(response)

	return
}
